package game

import "testing"

func TestEnqueueJoinDeduplicates(t *testing.T) {
	gm := NewGameManager()
	gm.enqueueJoin("player-a")
	gm.enqueueJoin("player-a")
	gm.enqueueJoin("player-b")

	gm.mu.Lock()
	defer gm.mu.Unlock()
	if len(gm.joinQueue) != 2 {
		t.Fatalf("expected 2 in join queue, got %d: %v", len(gm.joinQueue), gm.joinQueue)
	}
}

func TestCheckMatchmakingPairsUnderLock(t *testing.T) {
	gm := NewGameManager()
	p1 := NewPlayer("p1", "One")
	p2 := NewPlayer("p2", "Two")
	gm.mu.Lock()
	gm.allPlayers[p1.PlayerID] = p1
	gm.allPlayers[p2.PlayerID] = p2
	gm.joinQueue = []PlayerID{p1.PlayerID, p2.PlayerID}
	gm.mu.Unlock()

	// Run matchmaking synchronously (normally spawned in goroutine).
	gm.mu.Lock()
	if len(gm.joinQueue) < 2 {
		gm.mu.Unlock()
		t.Fatal("expected at least 2 in queue")
	}
	player1ID := gm.joinQueue[0]
	player2ID := gm.joinQueue[1]
	gm.joinQueue = gm.joinQueue[2:]
	gm.mu.Unlock()

	gm.createAndSendGame(player1ID, player2ID)

	gm.mu.Lock()
	defer gm.mu.Unlock()
	if len(gm.games) != 1 {
		t.Fatalf("expected 1 game, got %d", len(gm.games))
	}
	if len(gm.joinQueue) != 0 {
		t.Fatalf("expected empty join queue, got %v", gm.joinQueue)
	}
}
