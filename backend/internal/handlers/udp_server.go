package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/Nitin-Poojary/air-hockey/backend/internal/game"
	"github.com/Nitin-Poojary/air-hockey/backend/internal/utils"
	"golang.org/x/time/rate"
)

type UDPServer struct {
	gameManager *game.GameManager
	udpLimiters map[string]*rate.Limiter
	limitersMu  sync.Mutex
}

func NewUDPServer(gameManager *game.GameManager) *UDPServer {
	return &UDPServer{
		gameManager: gameManager,
		udpLimiters: make(map[string]*rate.Limiter),
	}
}

func (s *UDPServer) StartUDPServer() *net.UDPConn {
	addr := net.UDPAddr{
		Port: utils.GetEnvInt("UDPPORT", "8050"),
		IP:   net.ParseIP("0.0.0.0"),
	}
	conn, err := net.ListenUDP("udp", &addr)
	if err != nil {
		panic(err)
	}
	slog.Info("UDP server started", "port", addr.Port)
	return conn
}

func (s *UDPServer) StartUDPSession(conn *net.UDPConn) {
	buffer := make([]byte, 1024)
	for {
		n, clientAddr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			if strings.Contains(err.Error(), "use of closed network connection") {
				slog.Info("UDP connection closed, exiting session loop")
				return
			}
			slog.Error("Error reading from UDP", "error", err)
			continue
		}

		slog.Info("Received UDP packet", "bytes", n, "from", clientAddr.String(), "data", string(buffer[:n]))

		var move clientMovePacket
		if err := json.Unmarshal(buffer[:n], &move); err != nil {
			slog.Error("Error unmarshalling UDP packet", "error", err)
			continue
		}
		if move.GameID == "" || move.PlayerID == "" {
			continue
		}

		// Enforce rate limiting on paddle movement packets (drop silently if limit exceeded)
		if !s.allowUDP(string(move.GameID), string(move.PlayerID)) {
			continue
		}

		g := s.gameManager.GetGame(move.GameID)
		if g == nil {
			continue
		}
		if !g.MovePaddle(move.PlayerID, move.X, move.Y) {
			continue
		}
		g.SetPlayerAddress(move.PlayerID, clientAddr)
	}
}

func (s *UDPServer) allowUDP(gameID, playerID string) bool {
	s.limitersMu.Lock()
	defer s.limitersMu.Unlock()
	key := gameID + ":" + playerID
	limiter, exists := s.udpLimiters[key]
	if !exists {
		// Cap paddle packets ~60/sec, burst = 5
		limiter = rate.NewLimiter(rate.Limit(60.0), 5)
		s.udpLimiters[key] = limiter
	}
	return limiter.Allow()
}

func (s *UDPServer) cleanupGameLimiters(gameID game.GameID) {
	s.limitersMu.Lock()
	defer s.limitersMu.Unlock()
	prefix := string(gameID) + ":"
	for key := range s.udpLimiters {
		if strings.HasPrefix(key, prefix) {
			delete(s.udpLimiters, key)
		}
	}
}

func (s *UDPServer) StartTickingOnGameCreation(ctx context.Context, conn *net.UDPConn) {
	gameChan := s.gameManager.RecvGameCreation()
	for {
		select {
		case g, ok := <-gameChan:
			if !ok {
				slog.Info("Game creation channel closed, exiting ticker loop")
				return
			}
			go s.StartGameSession(g, conn)
		case <-ctx.Done():
			slog.Info("UDP StartTickingOnGameCreation loop exiting due to context cancel")
			return
		}
	}
}

func (s *UDPServer) StartGameSession(g *game.Game, conn *net.UDPConn) {
	defer s.cleanupGameLimiters(g.ID)

	fps := utils.GetEnvInt("TICKRATE", "30")
	ticker := time.NewTicker(time.Second / time.Duration(fps))
	defer ticker.Stop()

	ctx := g.Context()
	physics := g.Physics
	slog.Info("Starting game session", "gameID", g.ID)

	for {
		select {
		case <-ctx.Done():
			g.State.Close()
			s.gameManager.RemoveGame(g.ID)
			return

		case winner := <-g.State.RecvRoundWinner():
			slog.Info("Round finished", "gameID", g.ID, "winnerID", winner)

			winnerPlayer, ok := s.gameManager.GetPlayer(winner)
			if ok && winnerPlayer != nil {
				g.State.SetRoundWinnerName(winnerPlayer.DisplayName)
			}

			s.broadcastStateSnapshot(g, conn)
			g.ScheduleRoundClear(time.Second)

		case winner := <-g.State.RecvGameWinner():
			slog.Info("Game finished", "gameID", g.ID, "winnerID", winner, "event", "game_end")
			s.broadcastMatchResult(g, conn, winner)
			g.State.Close()
			s.gameManager.RemoveGame(g.ID)
			return

		case <-ticker.C:
			physics.Tick()
			s.broadcastStateSnapshot(g, conn)
		}
	}
}

func (s *UDPServer) broadcastStateSnapshot(g *game.Game, conn *net.UDPConn) {
	snapshot := g.State.Snapshot()
	stateBytes, err := json.Marshal(snapshot)
	if err != nil {
		return
	}
	g.ForEachAddress(func(addr *net.UDPAddr) {
		_, _ = conn.WriteToUDP(stateBytes, addr)
	})
}

func (s *UDPServer) broadcastMatchResult(g *game.Game, conn *net.UDPConn, winner game.PlayerID) {
	winnerPlayer, ok := s.gameManager.GetPlayer(winner)
	var winnerName string
	if ok && winnerPlayer != nil {
		winnerName = winnerPlayer.DisplayName
	}
	matchResult := matchResultPacket{
		GameID:     g.ID,
		Winner:     winner,
		WinnerID:   winner,
		WinnerName: winnerName,
	}
	matchResultBytes, err := json.Marshal(matchResult)
	if err != nil {
		return
	}
	g.ForEachAddress(func(addr *net.UDPAddr) {
		_, _ = conn.WriteToUDP(matchResultBytes, addr)
	})
	slog.Info("Match finished", "gameID", g.ID, "winnerName", winnerName, "winnerID", winner, "event", "game_end")
}
