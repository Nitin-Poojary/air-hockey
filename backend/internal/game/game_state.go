package game

import (
	"log/slog"
	"sync"
)

type PlayerID string

type GameState struct {
	Player1         *Paddle `json:"player1"`
	Player2         *Paddle `json:"player2"`
	Score           *Score  `json:"score"`
	Puck            *Puck   `json:"puck"`
	Board           *Board  `json:"board"`
	roundWinnerChan chan PlayerID
	gameWinnerChan  chan PlayerID
	RoundEnded      bool     `json:"roundEnded"`
	RoundWinner     PlayerID `json:"roundWinner"`
	RoundWinnerName string   `json:"roundWinnerName"`
	mu              sync.Mutex
}

// GameStateSnapshot is an immutable copy safe for JSON marshaling off the tick loop.
type GameStateSnapshot struct {
	Player1         Paddle   `json:"player1"`
	Player2         Paddle   `json:"player2"`
	Score           Score    `json:"score"`
	Puck            Puck     `json:"puck"`
	Board           Board    `json:"board"`
	RoundEnded      bool     `json:"roundEnded"`
	RoundWinner     PlayerID `json:"roundWinner"`
	RoundWinnerName string   `json:"roundWinnerName"`
}

func NewGameState(player1Id, player2Id PlayerID) *GameState {
	return &GameState{
		Player1:         NewPaddle(player1Id, BoardWidth/2, PaddleRadius),
		Player2:         NewPaddle(player2Id, BoardWidth/2, BoardHeight-PaddleRadius),
		Score:           NewScore(),
		Puck:            NewPuck(BoardWidth/2, BoardHeight/2),
		Board:           NewBoard(),
		roundWinnerChan: make(chan PlayerID, 1),
		gameWinnerChan:  make(chan PlayerID, 1),
	}
}

func (gs *GameState) Snapshot() GameStateSnapshot {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	return GameStateSnapshot{
		Player1:         copyPaddle(*gs.Player1),
		Player2:         copyPaddle(*gs.Player2),
		Score:           *gs.Score,
		Puck:            *gs.Puck,
		Board:           *gs.Board,
		RoundEnded:      gs.RoundEnded,
		RoundWinner:     gs.RoundWinner,
		RoundWinnerName: gs.RoundWinnerName,
	}
}

func copyPaddle(p Paddle) Paddle {
	return Paddle{
		PlayerID: p.PlayerID,
		Pos:      &Position{X: p.Pos.X, Y: p.Pos.Y},
		Velocity: &Vector{VX: p.Velocity.VX, VY: p.Velocity.VY},
		Radius:   p.Radius,
	}
}

func (gs *GameState) ResetGameRound() {
	gs.mu.Lock()
	defer gs.mu.Unlock()

	gs.Player1.Pos.X = BoardWidth / 2
	gs.Player1.Pos.Y = PaddleRadius
	gs.Player1.Velocity.VX = 0
	gs.Player1.Velocity.VY = 0

	gs.Player2.Pos.X = BoardWidth / 2
	gs.Player2.Pos.Y = BoardHeight - PaddleRadius
	gs.Player2.Velocity.VX = 0
	gs.Player2.Velocity.VY = 0

	gs.Puck.Pos.X = BoardWidth / 2
	gs.Puck.Pos.Y = BoardHeight / 2
	gs.Puck.Velocity.VX = 0
	gs.Puck.Velocity.VY = 0
}

func (gs *GameState) MovePaddle(playerID PlayerID, newX, newY float64) {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	if gs.RoundEnded {
		return
	}
	paddle := gs.paddleLocked(playerID)
	if paddle == nil {
		return
	}
	newX, newY = clampPaddlePosition(playerID, newX, newY, gs)
	paddle.Velocity.VX = newX - paddle.Pos.X
	paddle.Velocity.VY = newY - paddle.Pos.Y
	paddle.Pos.X = newX
	paddle.Pos.Y = newY
}

func clampPaddlePosition(playerID PlayerID, x, y float64, gs *GameState) (float64, float64) {
	r := PaddleRadius
	w := gs.Board.Width
	h := gs.Board.Height
	x = clamp(x, r, w-r)
	switch playerID {
	case gs.Player1.PlayerID:
		y = clamp(y, r, h*0.48)
	case gs.Player2.PlayerID:
		y = clamp(y, h*0.52, h-r)
	default:
		y = clamp(y, r, h-r)
	}
	return x, y
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// paddleLocked assumes gs.mu is held by the caller — do not call this without locking first
func (gs *GameState) paddleLocked(playerID PlayerID) *Paddle {
	switch playerID {
	case gs.Player1.PlayerID:
		return gs.Player1
	case gs.Player2.PlayerID:
		return gs.Player2
	default:
		return nil
	}
}

func (gs *GameState) IncrementScore(playerID PlayerID) {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	switch playerID {
	case gs.Player1.PlayerID:
		gs.Score.Player1++
	case gs.Player2.PlayerID:
		gs.Score.Player2++
	}
}

func (gs *GameState) RecvRoundWinner() <-chan PlayerID {
	return gs.roundWinnerChan
}

func (gs *GameState) ClearRoundEnded() {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	gs.RoundEnded = false
	gs.RoundWinner = ""
	gs.RoundWinnerName = ""
}

func (gs *GameState) SetGameWinner(playerID PlayerID) {
	slog.Info("SetGameWinner called", "playerID", playerID)
	gs.gameWinnerChan <- playerID
}

func (gs *GameState) RecvGameWinner() <-chan PlayerID {
	return gs.gameWinnerChan
}

func (gs *GameState) SetRoundWinnerName(name string) {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	gs.RoundWinnerName = name
}

func (gs *GameState) Close() {
	close(gs.roundWinnerChan)
	close(gs.gameWinnerChan)
}
