package handlers

import "github.com/Nitin-Poojary/air-hockey/backend/internal/game"

type joinRequest struct {
	PlayerID game.PlayerID `json:"playerID"`
}

type leaveRequest struct {
	PlayerID game.PlayerID `json:"playerID"`
}

type unregisterRequest struct {
	PlayerID game.PlayerID `json:"playerID"`
}

type removeGameRequest struct {
	GameID game.GameID `json:"gameID"`
}

type removePlayerFromGameRequest struct {
	GameID   game.GameID   `json:"gameID"`
	PlayerID game.PlayerID `json:"playerID"`
}

type clientMovePacket struct {
	GameID   game.GameID   `json:"gameID"`
	PlayerID game.PlayerID `json:"playerID"`
	X        float64       `json:"x"`
	Y        float64       `json:"y"`
}
type matchResultPacket struct {
	GameID     game.GameID   `json:"gameID"`
	Winner     game.PlayerID `json:"winner"`
	WinnerID   game.PlayerID `json:"winnerID"`
	WinnerName string        `json:"winnerName"`
}

type resolveRequest struct {
	Name string `json:"name"`
}

type resolveResponse struct {
	PlayerID    game.PlayerID `json:"playerID"`
	DisplayName string        `json:"displayName"`
}
