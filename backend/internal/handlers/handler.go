package handlers

import (
	"log/slog"
	"net"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/Nitin-Poojary/air-hockey/backend/internal/game"
	"github.com/Nitin-Poojary/air-hockey/backend/internal/store"
	"github.com/google/uuid"
	"golang.org/x/time/rate"
)

var playerNamePattern = regexp.MustCompile(`^[a-zA-Z0-9 _-]+$`)

type Handler struct {
	gameManager    *game.GameManager
	redisStore     store.PlayerStore
	resolveLimiter *KeyLimiter
	joinLimiter    *KeyLimiter
	leaveLimiter   *KeyLimiter
}

func NewHandler(gameManager *game.GameManager, redisStore store.PlayerStore) *Handler {
	return &Handler{
		gameManager:    gameManager,
		redisStore:     redisStore,
		resolveLimiter: NewKeyLimiter(rate.Limit(10.0/60.0), 10), // 10/min
		joinLimiter:    NewKeyLimiter(rate.Limit(3.0/10.0), 3),   // 3/10s
		leaveLimiter:   NewKeyLimiter(rate.Limit(10.0/60.0), 10), // 10/min
	}
}

func (h *Handler) HandleResolve(w http.ResponseWriter, r *http.Request) {
	// 1. IP rate limiting
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		ip = r.RemoteAddr
	}
	if !h.resolveLimiter.GetLimiter(ip).Allow() {
		slog.Warn("Rate limit exceeded for resolve", "ip", ip, "event", "rate_limited")
		WriteResponse(w, map[string]string{"error": "rate limit exceeded"}, http.StatusTooManyRequests)
		return
	}

	var req resolveRequest
	if err := ReadRequest(r, &req); err != nil {
		WriteResponse(w, map[string]string{"error": "invalid request body"}, http.StatusBadRequest)
		return
	}

	name := strings.TrimSpace(req.Name)
	if len(name) < 2 || len(name) > 20 {
		WriteResponse(w, map[string]string{"error": "name must be between 2 and 20 characters"}, http.StatusBadRequest)
		return
	}

	if !playerNamePattern.MatchString(name) {
		WriteResponse(w, map[string]string{"error": "name contains invalid characters"}, http.StatusBadRequest)
		return
	}

	normalized := strings.ToLower(name)
	ctx := r.Context()

	// 2. Lookup in Redis
	playerIDStr, err := h.redisStore.GetPlayerIDByName(ctx, normalized)
	if err != nil {
		slog.Error("Redis error in HandleResolve lookup", "error", err)
		WriteResponse(w, map[string]string{"error": "internal store error"}, http.StatusInternalServerError)
		return
	}

	var playerID game.PlayerID
	var resolvedName string

	if playerIDStr != "" {
		// Found existing player
		playerID = game.PlayerID(playerIDStr)
		origName, err := h.redisStore.GetPlayerNameByID(ctx, playerIDStr)
		if err == nil && origName != "" {
			resolvedName = origName
		} else {
			resolvedName = name
		}
	} else {
		// Generate new player ID
		newID := uuid.New().String()
		playerID = game.PlayerID(newID)
		resolvedName = name

		// Try to register in Redis
		success, err := h.redisStore.RegisterPlayerName(ctx, normalized, newID, name)
		if err != nil {
			slog.Error("Redis RegisterPlayerName error", "error", err)
			WriteResponse(w, map[string]string{"error": "internal store error"}, http.StatusInternalServerError)
			return
		}

		if !success {
			// Lost registration race; reload winner's ID
			playerIDStr, err = h.redisStore.GetPlayerIDByName(ctx, normalized)
			if err != nil || playerIDStr == "" {
				WriteResponse(w, map[string]string{"error": "internal store error after registration race"}, http.StatusInternalServerError)
				return
			}
			playerID = game.PlayerID(playerIDStr)
			origName, err := h.redisStore.GetPlayerNameByID(ctx, playerIDStr)
			if err == nil && origName != "" {
				resolvedName = origName
			} else {
				resolvedName = name
			}
		}
	}

	// 3. Register/activate in GameManager in-memory map
	h.gameManager.RegisterPlayer(playerID, resolvedName)
	slog.Info("Player resolved", "playerID", playerID, "displayName", resolvedName, "event", "resolved")

	// 4. Respond
	WriteResponse(w, resolveResponse{
		PlayerID:    playerID,
		DisplayName: resolvedName,
	}, http.StatusOK)
}

func (h *Handler) HandleGetPlayer(w http.ResponseWriter, r *http.Request) {
	playerIDStr := r.PathValue("playerID")
	if playerIDStr == "" {
		WriteResponse(w, map[string]string{"error": ErrInvalidPlayerID}, http.StatusBadRequest)
		return
	}

	playerID := game.PlayerID(playerIDStr)

	// Check if already in memory
	player, ok := h.gameManager.GetPlayer(playerID)
	if ok {
		WriteResponse(w, resolveResponse{
			PlayerID:    player.PlayerID,
			DisplayName: player.DisplayName,
		}, http.StatusOK)
		return
	}

	// Check Redis if missing from memory
	ctx := r.Context()
	name, err := h.redisStore.GetPlayerNameByID(ctx, playerIDStr)
	if err == nil && name != "" {
		// Re-register in memory
		h.gameManager.RegisterPlayer(playerID, name)
		WriteResponse(w, resolveResponse{
			PlayerID:    playerID,
			DisplayName: name,
		}, http.StatusOK)
		return
	}

	WriteResponse(w, map[string]string{"error": ErrPlayerNotFound}, http.StatusNotFound)
}

func (h *Handler) HandleMatchStatus(w http.ResponseWriter, r *http.Request) {
	rawPlayerId := r.URL.Query().Get("playerID")
	if rawPlayerId == "" {
		WriteResponse(w, map[string]string{"error": ErrInvalidPlayerID}, http.StatusBadRequest)
		return
	}

	playerID := game.PlayerID(rawPlayerId)
	player, ok := h.gameManager.GetPlayer(playerID)
	
	if !ok {
		WriteResponse(w, map[string]string{"error": ErrInvalidPlayerID}, http.StatusBadRequest)
		return
	}

	// Enforce 1 concurrent long poll per player
	if !player.TryStartPoll() {
		slog.Warn("Rate limit exceeded: concurrent matchStatus poll", "playerID", playerID, "event", "rate_limited")
		WriteResponse(w, map[string]string{"error": "rate limit exceeded"}, http.StatusTooManyRequests)
		return
	}
	defer player.EndPoll()

	ctx := r.Context()  
	select {
	case result := <-player.RecvMatchReady():
		h.gameManager.RemovePlayerFromJoinQueue(playerID)
		WriteResponse(w, result, http.StatusOK)
	case <-time.After(90 * time.Second):
		h.gameManager.RemovePlayerFromJoinQueue(playerID)
		slog.Info("Player matchStatus timeout", "playerID", playerID)
		WriteResponse(w, map[string]string{"status": "timeout"}, http.StatusOK)
	case <-ctx.Done():
		h.gameManager.RemovePlayerFromJoinQueue(playerID)
		slog.Info("Player matchStatus request cancelled", "playerID", playerID)
		WriteResponse(w, map[string]string{"status": "request cancelled"}, 408)
	}
}

func (h *Handler) HandleJoin(w http.ResponseWriter, r *http.Request) {
	var joinRequest joinRequest

	if err := ReadRequest(r, &joinRequest); err != nil {
		WriteResponse(w, map[string]string{"error": ErrInvalidPlayerID}, http.StatusBadRequest)
		return
	}

	if joinRequest.PlayerID == "" {
		WriteResponse(w, map[string]string{"error": ErrInvalidPlayerID}, http.StatusBadRequest)
		return
	}
	
	playerID := game.PlayerID(joinRequest.PlayerID)

	// Idempotency: if already in queue, return 200 OK immediately without consuming rate limit
	if h.gameManager.IsPlayerInJoinQueue(playerID) {
		slog.Info("Player already in join queue (idempotent)", "playerID", playerID)
		WriteResponse(w, "Player joined the queue", http.StatusOK)
		return
	}

	// Rate limiting check
	if !h.joinLimiter.GetLimiter(string(playerID)).Allow() {
		slog.Warn("Rate limit exceeded for join", "playerID", playerID, "event", "rate_limited")
		WriteResponse(w, map[string]string{"error": "rate limit exceeded"}, http.StatusTooManyRequests)
		return
	}

	h.gameManager.JoinPlayer(playerID)
	slog.Info("Player joined queue", "playerID", playerID, "event", "joined")
	WriteResponse(w, "Player joined the queue", http.StatusOK)
}

func (h *Handler) HandleLeave(w http.ResponseWriter, r *http.Request) {
	var leaveRequest leaveRequest

	if err := ReadRequest(r, &leaveRequest); err != nil {
		WriteResponse(w, map[string]string{"error": ErrInvalidPlayerID}, http.StatusBadRequest)
		return
	}

	if leaveRequest.PlayerID == "" {
		WriteResponse(w, map[string]string{"error": ErrInvalidPlayerID}, http.StatusBadRequest)
		return
	}

	playerID := game.PlayerID(leaveRequest.PlayerID)

	// Rate limiting check
	if !h.leaveLimiter.GetLimiter(string(playerID)).Allow() {
		slog.Warn("Rate limit exceeded for leave", "playerID", playerID, "event", "rate_limited")
		WriteResponse(w, map[string]string{"error": "rate limit exceeded"}, http.StatusTooManyRequests)
		return
	}

	h.gameManager.LeavePlayer(playerID)
	slog.Info("Player left queue", "playerID", playerID, "event", "left")
	WriteResponse(w, "Player left the queue", http.StatusOK)
}

func (h *Handler) HandleUnregister(w http.ResponseWriter, r *http.Request) {
	var unregisterRequest unregisterRequest

	if err := ReadRequest(r, &unregisterRequest); err != nil {
		WriteResponse(w, map[string]string{"error": ErrInvalidPlayerID}, http.StatusBadRequest)
		return
	}

	if unregisterRequest.PlayerID == "" {
		WriteResponse(w, map[string]string{"error": ErrInvalidPlayerID}, http.StatusBadRequest)
		return
	}
	
	playerID := game.PlayerID(unregisterRequest.PlayerID)

	// Rate limiting check
	if !h.leaveLimiter.GetLimiter(string(playerID)).Allow() {
		slog.Warn("Rate limit exceeded for unregister", "playerID", playerID, "event", "rate_limited")
		WriteResponse(w, map[string]string{"error": "rate limit exceeded"}, http.StatusTooManyRequests)
		return
	}
	
	h.gameManager.UnregisterPlayer(playerID)
	slog.Info("Player unregistered", "playerID", playerID, "event", "unregister")
	WriteResponse(w, "Player unregistered", http.StatusOK)
}
