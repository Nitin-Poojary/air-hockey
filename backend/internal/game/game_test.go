package game

import (
	"context"
	"testing"
	"time"
)

func TestJoinQueuePairsTwoPlayers(t *testing.T) {
	gm := NewGameManager()
	defer gm.Close()
	go gm.HandleJoinQueue()

	p1 := gm.RegisterPlayer("player-1", "Player One")
	p2 := gm.RegisterPlayer("player-2", "Player Two")

	gm.JoinPlayer(p1.PlayerID)
	gm.JoinPlayer(p2.PlayerID)

	select {
	case game := <-gm.RecvGameCreation():
		if game == nil {
			t.Fatal("expected non-nil game")
		}
		if _, ok := game.Players[p1.PlayerID]; !ok {
			t.Errorf("expected player 1 in game")
		}
		if _, ok := game.Players[p2.PlayerID]; !ok {
			t.Errorf("expected player 2 in game")
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timeout waiting for matchmaking")
	}
}

func TestLeaveBeforeMatchRemovesPlayer(t *testing.T) {
	gm := NewGameManager()
	defer gm.Close()
	go gm.HandleJoinQueue()
	go gm.HandleLeaveQueue()

	p1 := gm.RegisterPlayer("player-1", "Player One")
	p2 := gm.RegisterPlayer("player-2", "Player Two")

	gm.JoinPlayer(p1.PlayerID)
	// Small delay to ensure player 1 is in the queue
	time.Sleep(10 * time.Millisecond)

	gm.LeavePlayer(p1.PlayerID)
	// Small delay to ensure leave is processed
	time.Sleep(10 * time.Millisecond)

	gm.JoinPlayer(p2.PlayerID)

	select {
	case <-gm.RecvGameCreation():
		t.Fatal("expected no game to be created since player 1 left")
	case <-time.After(200 * time.Millisecond):
		// Success: no game created
	}
}

func TestMovePaddleIgnoredWhenRoundEnded(t *testing.T) {
	p1 := NewPlayer("player-1", "Player One")
	p2 := NewPlayer("player-2", "Player Two")
	g := NewGame(context.Background(), p1, p2)

	// Round ended is false initially. Move paddle should succeed.
	oldX, oldY := g.State.Player1.Pos.X, g.State.Player1.Pos.Y
	moved := g.MovePaddle(p1.PlayerID, oldX+10, oldY+10)
	if !moved {
		t.Error("expected MovePaddle to succeed")
	}
	if g.State.Player1.Pos.X == oldX || g.State.Player1.Pos.Y == oldY {
		t.Error("expected paddle position to update")
	}

	// Set round ended to true
	g.State.mu.Lock()
	g.State.RoundEnded = true
	g.State.mu.Unlock()

	lastX, lastY := g.State.Player1.Pos.X, g.State.Player1.Pos.Y
	// Try to move again
	g.MovePaddle(p1.PlayerID, lastX+10, lastY+10)
	if g.State.Player1.Pos.X != lastX || g.State.Player1.Pos.Y != lastY {
		t.Error("expected paddle position update to be ignored when round ended")
	}
}

func TestGoalIncrementsScoreAndEndsMatch(t *testing.T) {
	p1 := NewPlayer("player-1", "Player One")
	p2 := NewPlayer("player-2", "Player Two")

	t.Run("Goal increments score and triggers round end", func(t *testing.T) {
		g := NewGame(context.Background(), p1, p2)
		physics := NewGamePhysics(g.State)

		// Place puck near top goal moving upwards
		g.State.mu.Lock()
		g.State.Puck.Pos.X = 200 // Goal is between 175 and 375
		g.State.Puck.Pos.Y = 5
		g.State.Puck.Velocity.VX = 0
		g.State.Puck.Velocity.VY = -10
		g.State.mu.Unlock()

		physics.Tick()

		// Player 2 score should increment
		g.State.mu.Lock()
		p2Score := g.State.Score.Player2
		roundEnded := g.State.RoundEnded
		roundWinner := g.State.RoundWinner
		g.State.mu.Unlock()

		if p2Score != 1 {
			t.Errorf("expected Player 2 score to be 1, got %d", p2Score)
		}
		if !roundEnded {
			t.Error("expected round to be marked as ended")
		}
		if roundWinner != p2.PlayerID {
			t.Errorf("expected round winner to be %s, got %s", p2.PlayerID, roundWinner)
		}

		// Check round winner chan received
		select {
		case winner := <-g.State.RecvRoundWinner():
			if winner != p2.PlayerID {
				t.Errorf("expected round winner from channel to be %s, got %s", p2.PlayerID, winner)
			}
		default:
			t.Error("expected winner notification on roundWinnerChan")
		}
	})

	t.Run("Goal ends match at MaxScoreToWin", func(t *testing.T) {
		g := NewGame(context.Background(), p1, p2)
		physics := NewGamePhysics(g.State)

		// Set score to MaxScoreToWin - 1
		g.State.mu.Lock()
		g.State.Score.Player2 = MaxScoreToWin - 1
		g.State.Puck.Pos.X = 200
		g.State.Puck.Pos.Y = 5
		g.State.Puck.Velocity.VX = 0
		g.State.Puck.Velocity.VY = -10
		g.State.mu.Unlock()

		physics.Tick()

		g.State.mu.Lock()
		p2Score := g.State.Score.Player2
		g.State.mu.Unlock()

		if p2Score != MaxScoreToWin {
			t.Errorf("expected Player 2 score to reach MaxScoreToWin (%d), got %d", MaxScoreToWin, p2Score)
		}

		// Check game winner chan received
		select {
		case winner := <-g.State.RecvGameWinner():
			if winner != p2.PlayerID {
				t.Errorf("expected game winner from channel to be %s, got %s", p2.PlayerID, winner)
			}
		case <-time.After(100 * time.Millisecond):
			t.Error("expected game winner notification on gameWinnerChan, timed out")
		}
	})
}
