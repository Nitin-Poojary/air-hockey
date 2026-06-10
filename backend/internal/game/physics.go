package game

import "math"

type GamePhysics struct {
	GameState *GameState
}

func NewGamePhysics(gs *GameState) *GamePhysics {
	return &GamePhysics{GameState: gs}
}

// Tick runs one frame of physics. Winner notifications are sent after releasing the state lock.
func (gp *GamePhysics) Tick() {
	gs := gp.GameState
	gs.mu.Lock()
	if gs.RoundEnded {
		gs.mu.Unlock()
		return
	}

	puck := gs.Puck
	puck.Pos.X += puck.Velocity.VX
	puck.Pos.Y += puck.Velocity.VY

	gp.checkCollisionsAndUpdatePuckLocked(gs, gs.Player1)
	gp.checkCollisionsAndUpdatePuckLocked(gs, gs.Player2)

	gs.Player1.Velocity.VX = 0
	gs.Player1.Velocity.VY = 0
	gs.Player2.Velocity.VX = 0
	gs.Player2.Velocity.VY = 0

	roundWinner, gameWinner := gp.handleGoalsLocked(gs)
	gs.mu.Unlock()

	if gameWinner != "" {
		gs.gameWinnerChan <- gameWinner
		return
	}
	if roundWinner != "" {
		gs.roundWinnerChan <- roundWinner
	}
}

func (gp *GamePhysics) checkCollisionsAndUpdatePuckLocked(gs *GameState, paddle *Paddle) {
	puck := gs.Puck
	width := gs.Board.Width
	height := gs.Board.Height

	dx := puck.Pos.X - paddle.Pos.X
	dy := puck.Pos.Y - paddle.Pos.Y
	distance := math.Sqrt(dx*dx + dy*dy)
	minDistance := puck.Radius + paddle.Radius

	if distance < minDistance && distance != 0 {
		nx := dx / distance
		ny := dy / distance
		overlap := minDistance - distance
		puck.Pos.X += nx * overlap
		puck.Pos.Y += ny * overlap

		relVX := puck.Velocity.VX - paddle.Velocity.VX
		relVY := puck.Velocity.VY - paddle.Velocity.VY
		dot := relVX*nx + relVY*ny
		if dot < 0 {
			restitution := 1.1
			j := -(1 + restitution) * dot
			puck.Velocity.VX += j * nx
			puck.Velocity.VY += j * ny
		}
	}

	if puck.Pos.X-puck.Radius < 0 {
		puck.Pos.X = puck.Radius
		puck.Velocity.VX *= -1
	}
	if puck.Pos.X+puck.Radius > width {
		puck.Pos.X = width - puck.Radius
		puck.Velocity.VX *= -1
	}
	if puck.Pos.Y-puck.Radius < 0 && !isPuckInGoal(puck) {
		puck.Pos.Y = puck.Radius
		puck.Velocity.VY *= -1
	}
	if puck.Pos.Y+puck.Radius > height && !isPuckInGoal(puck) {
		puck.Pos.Y = height - puck.Radius
		puck.Velocity.VY *= -1
	}

	gp.capPuckSpeedLocked(puck, MaxPuckSpeed)
}

// handleGoalsLocked returns roundWinner and/or gameWinner player IDs (at most one non-empty).
func (gp *GamePhysics) handleGoalsLocked(gs *GameState) (roundWinner, gameWinner PlayerID) {
	puck := gs.Puck
	height := gs.Board.Height

	if puck.Pos.Y-puck.Radius < 0 && isPuckInGoal(puck) {
		gs.Score.Player2++
		gp.resetGameRoundLocked(gs)
		if gp.checkGameEndLocked(gs) {
			return "", gs.Player2.PlayerID
		}
		gs.RoundEnded = true
		gs.RoundWinner = gs.Player2.PlayerID
		return gs.Player2.PlayerID, ""
	}

	if puck.Pos.Y+puck.Radius > height && isPuckInGoal(puck) {
		gs.Score.Player1++
		gp.resetGameRoundLocked(gs)
		if gp.checkGameEndLocked(gs) {
			return "", gs.Player1.PlayerID
		}
		gs.RoundEnded = true
		gs.RoundWinner = gs.Player1.PlayerID
		return gs.Player1.PlayerID, ""
	}

	return "", ""
}

func (gp *GamePhysics) resetGameRoundLocked(gs *GameState) {
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

func isPuckInGoal(puck *Puck) bool {
	return puck.Pos.X > 175 && puck.Pos.X < 375
}

func (gp *GamePhysics) checkGameEndLocked(gs *GameState) bool {
	score := gs.Score
	if score.Player1 >= MaxScoreToWin && score.Player1 > score.Player2 {
		return true
	}
	if score.Player2 >= MaxScoreToWin && score.Player2 > score.Player1 {
		return true
	}
	return false
}

func (gp *GamePhysics) capPuckSpeedLocked(puck *Puck, max float64) {
	speed := math.Sqrt(
		puck.Velocity.VX*puck.Velocity.VX +
			puck.Velocity.VY*puck.Velocity.VY,
	)
	if speed <= max {
		return
	}
	scale := max / speed
	puck.Velocity.VX *= scale
	puck.Velocity.VY *= scale
}
