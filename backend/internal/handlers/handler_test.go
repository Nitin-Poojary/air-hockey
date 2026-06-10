package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/Nitin-Poojary/air-hockey/backend/internal/game"
)

// MockPlayerStore is an in-memory implementation of store.PlayerStore for testing.
type MockPlayerStore struct {
	IDByName   map[string]string
	NameByID   map[string]string
	RegErr     error
	GetIDErr   error
	GetNameErr error
	mu         sync.Mutex
}

func NewMockPlayerStore() *MockPlayerStore {
	return &MockPlayerStore{
		IDByName: make(map[string]string),
		NameByID: make(map[string]string),
	}
}

func (m *MockPlayerStore) GetPlayerIDByName(ctx context.Context, normalized string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.GetIDErr != nil {
		return "", m.GetIDErr
	}
	return m.IDByName[normalized], nil
}

func (m *MockPlayerStore) GetPlayerNameByID(ctx context.Context, playerID string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.GetNameErr != nil {
		return "", m.GetNameErr
	}
	return m.NameByID[playerID], nil
}

func (m *MockPlayerStore) RegisterPlayerName(ctx context.Context, normalized string, playerID string, originalName string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.RegErr != nil {
		return false, m.RegErr
	}
	if _, exists := m.IDByName[normalized]; exists {
		return false, nil
	}
	m.IDByName[normalized] = playerID
	m.NameByID[playerID] = originalName
	return true, nil
}

func TestHandleResolve(t *testing.T) {
	gameManager := game.NewGameManager()
	defer gameManager.Close()
	go gameManager.Start()

	mockStore := NewMockPlayerStore()
	handler := NewHandler(gameManager, mockStore)

	// 1. First registration
	reqBody, _ := json.Marshal(resolveRequest{Name: "TestPlayer"})
	req := httptest.NewRequest("POST", "/players/resolve", bytes.NewReader(reqBody))
	rec := httptest.NewRecorder()

	handler.HandleResolve(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp resolveResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.DisplayName != "TestPlayer" {
		t.Errorf("expected DisplayName 'TestPlayer', got '%s'", resp.DisplayName)
	}

	if resp.PlayerID == "" {
		t.Error("expected non-empty PlayerID")
	}

	// 2. Resolve same name again (simulate reinstall/recovery) - should return SAME ID
	req2 := httptest.NewRequest("POST", "/players/resolve", bytes.NewReader(reqBody))
	rec2 := httptest.NewRecorder()
	handler.HandleResolve(rec2, req2)

	if rec2.Code != http.StatusOK {
		t.Errorf("expected status 200 on second resolve, got %d", rec2.Code)
	}

	var resp2 resolveResponse
	_ = json.Unmarshal(rec2.Body.Bytes(), &resp2)

	if resp2.PlayerID != resp.PlayerID {
		t.Errorf("expected same PlayerID %s, got %s", resp.PlayerID, resp2.PlayerID)
	}
}

func TestHandleResolveValidation(t *testing.T) {
	gameManager := game.NewGameManager()
	defer gameManager.Close()
	go gameManager.Start()

	mockStore := NewMockPlayerStore()
	handler := NewHandler(gameManager, mockStore)

	tests := []struct {
		name       string
		inputName  string
		expectCode int
	}{
		{"Short Name", "A", http.StatusBadRequest},
		{"Long Name", "ThisIsAVeryLongNameThatExceedsTwentyCharactersLimit", http.StatusBadRequest},
		{"Special Characters", "Player@123", http.StatusBadRequest},
		{"Valid Spaces and Hyphens", "Player-1 New", http.StatusOK},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			reqBody, _ := json.Marshal(resolveRequest{Name: tc.inputName})
			req := httptest.NewRequest("POST", "/players/resolve", bytes.NewReader(reqBody))
			rec := httptest.NewRecorder()

			handler.HandleResolve(rec, req)

			if rec.Code != tc.expectCode {
				t.Errorf("for input %q: expected code %d, got %d: %s", tc.inputName, tc.expectCode, rec.Code, rec.Body.String())
			}
		})
	}
}

func TestHandleGetPlayer(t *testing.T) {
	gameManager := game.NewGameManager()
	defer gameManager.Close()

	mockStore := NewMockPlayerStore()
	handler := NewHandler(gameManager, mockStore)

	// Case 1: Player not found
	req := httptest.NewRequest("GET", "/players/nonexistent-id", nil)
	req.SetPathValue("playerID", "nonexistent-id")
	rec := httptest.NewRecorder()
	handler.HandleGetPlayer(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}

	// Case 2: Player exists in mock store (but not in memory yet)
	_, _ = mockStore.RegisterPlayerName(context.Background(), "existingplayer", "player-123", "ExistingPlayer")
	
	req2 := httptest.NewRequest("GET", "/players/player-123", nil)
	req2.SetPathValue("playerID", "player-123")
	rec2 := httptest.NewRecorder()
	handler.HandleGetPlayer(rec2, req2)

	if rec2.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec2.Code, rec2.Body.String())
	}

	var resp resolveResponse
	if err := json.Unmarshal(rec2.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if resp.PlayerID != "player-123" || resp.DisplayName != "ExistingPlayer" {
		t.Errorf("unexpected response: %+v", resp)
	}

	// Case 3: Player exists in memory
	// (Already registered in memory in case 2, let's verify it gets it from gameManager)
	req3 := httptest.NewRequest("GET", "/players/player-123", nil)
	req3.SetPathValue("playerID", "player-123")
	rec3 := httptest.NewRecorder()
	handler.HandleGetPlayer(rec3, req3)

	if rec3.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec3.Code)
	}
}

func TestJoinAndMatchStatusHappyPath(t *testing.T) {
	gameManager := game.NewGameManager()
	defer gameManager.Close()
	go gameManager.HandleJoinQueue()

	mockStore := NewMockPlayerStore()
	handler := NewHandler(gameManager, mockStore)

	// Register two players
	p1 := gameManager.RegisterPlayer("player-a", "Player A")
	p2 := gameManager.RegisterPlayer("player-b", "Player B")

	// 1. Join player A
	joinBodyA, _ := json.Marshal(joinRequest{PlayerID: p1.PlayerID})
	reqJoinA := httptest.NewRequest("POST", "/join", bytes.NewReader(joinBodyA))
	recJoinA := httptest.NewRecorder()
	handler.HandleJoin(recJoinA, reqJoinA)

	if recJoinA.Code != http.StatusOK {
		t.Fatalf("Join A failed: %d", recJoinA.Code)
	}

	// 2. Join player B
	joinBodyB, _ := json.Marshal(joinRequest{PlayerID: p2.PlayerID})
	reqJoinB := httptest.NewRequest("POST", "/join", bytes.NewReader(joinBodyB))
	recJoinB := httptest.NewRecorder()
	handler.HandleJoin(recJoinB, reqJoinB)

	if recJoinB.Code != http.StatusOK {
		t.Fatalf("Join B failed: %d", recJoinB.Code)
	}

	// Wait a moment for matchmaking goroutine to run
	time.Sleep(50 * time.Millisecond)

	// 3. Get match status for player A
	reqStatusA := httptest.NewRequest("GET", "/matchStatus?playerID=player-a", nil)
	recStatusA := httptest.NewRecorder()
	handler.HandleMatchStatus(recStatusA, reqStatusA)

	if recStatusA.Code != http.StatusOK {
		t.Errorf("expected status 200 for player A match status, got %d: %s", recStatusA.Code, recStatusA.Body.String())
	}

	var matchA game.MatchResult
	if err := json.Unmarshal(recStatusA.Body.Bytes(), &matchA); err != nil {
		t.Fatalf("failed to decode match A response: %v", err)
	}

	// 4. Get match status for player B
	reqStatusB := httptest.NewRequest("GET", "/matchStatus?playerID=player-b", nil)
	recStatusB := httptest.NewRecorder()
	handler.HandleMatchStatus(recStatusB, reqStatusB)

	if recStatusB.Code != http.StatusOK {
		t.Errorf("expected status 200 for player B match status, got %d: %s", recStatusB.Code, recStatusB.Body.String())
	}

	var matchB game.MatchResult
	if err := json.Unmarshal(recStatusB.Body.Bytes(), &matchB); err != nil {
		t.Fatalf("failed to decode match B response: %v", err)
	}

	if matchA.GameID != matchB.GameID {
		t.Errorf("expected same GameID, got %s and %s", matchA.GameID, matchB.GameID)
	}
}
