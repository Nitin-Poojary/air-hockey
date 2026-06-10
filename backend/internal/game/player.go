package game

import (
	"log/slog"
	"sync"
)

type Player struct {
	PlayerID    PlayerID `json:"player_id"`
	DisplayName string   `json:"display_name"`
	sendChan    chan []byte
	matchReady  chan MatchResult
	isPolling   bool
	pollMu      sync.Mutex
}

func NewPlayer(playerID PlayerID, displayName string) *Player {
	return &Player{
		PlayerID:    playerID,
		DisplayName: displayName,
		sendChan:    make(chan []byte),
		matchReady:  make(chan MatchResult, 1),
	}
}

func (c *Player) TryStartPoll() bool {
	c.pollMu.Lock()
	defer c.pollMu.Unlock()
	if c.isPolling {
		return false
	}
	c.isPolling = true
	return true
}

func (c *Player) EndPoll() {
	c.pollMu.Lock()
	defer c.pollMu.Unlock()
	c.isPolling = false
}

func (c *Player) Send(msg []byte) {
	c.sendChan <- msg
}

func (c *Player) Close() {
	close(c.sendChan)
	close(c.matchReady)
}

type MatchResult struct {
	GameID GameID            `json:"GameID"`
	State  GameStateSnapshot `json:"State"`
}

func NewMatchResult(gameID GameID, state GameStateSnapshot) MatchResult {
	return MatchResult{
		GameID: gameID,
		State:  state,
	}
}

func (c *Player) RecvMatchReady() <-chan MatchResult {
	return c.matchReady
}

func (c *Player) SendMatchReady(result MatchResult) {
	c.matchReady <- result
	slog.Info("Sending match ready event", "gameID", result.GameID)
}
