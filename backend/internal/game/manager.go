package game

import (
	"context"
	"log/slog"
	"slices"
	"sync"
)

const gameCreatedChanBuffer = 64

type GameManager struct {
	games           map[GameID]*Game
	allPlayers      map[PlayerID]*Player
	joinQueue       []PlayerID
	leaveQueue      []PlayerID
	Register        chan PlayerID
	Unregister      chan PlayerID
	joinChan        chan PlayerID
	leaveChan       chan PlayerID
	gameCreatedChan chan *Game
	parentCtx       context.Context
	mu              sync.Mutex
}

func NewGameManager() *GameManager {
	return NewGameManagerWithContext(context.Background())
}

func NewGameManagerWithContext(parent context.Context) *GameManager {
	if parent == nil {
		parent = context.Background()
	}
	return &GameManager{
		games:           make(map[GameID]*Game),
		allPlayers:      make(map[PlayerID]*Player),
		joinQueue:       make([]PlayerID, 0),
		leaveQueue:      make([]PlayerID, 0),
		Register:        make(chan PlayerID),
		Unregister:      make(chan PlayerID),
		joinChan:        make(chan PlayerID),
		leaveChan:       make(chan PlayerID),
		gameCreatedChan: make(chan *Game, gameCreatedChanBuffer),
		parentCtx:       parent,
	}
}

func (gm *GameManager) Start() {
	slog.Info("Game manager started")
	for {
		select {
		case playerID := <-gm.Register:
			gm.RegisterPlayer(playerID, "")
		case playerID := <-gm.Unregister:
			gm.UnregisterPlayer(playerID)
		case <-gm.parentCtx.Done():
			slog.Info("Game manager Start loop exiting")
			return
		}
	}
}

func (gm *GameManager) GetPlayer(playerID PlayerID) (*Player, bool) {
	gm.mu.Lock()
	defer gm.mu.Unlock()
	p, ok := gm.allPlayers[playerID]
	return p, ok
}

func (gm *GameManager) RegisterPlayer(playerID PlayerID, displayName string) *Player {
	gm.mu.Lock()
	defer gm.mu.Unlock()
	if p, exists := gm.allPlayers[playerID]; exists {
		if displayName != "" {
			p.DisplayName = displayName
		}
		return p
	}
	p := NewPlayer(playerID, displayName)
	gm.allPlayers[playerID] = p
	slog.Info("Player registered", "playerID", playerID, "displayName", displayName)
	return p
}

func (gm *GameManager) UnregisterPlayer(playerID PlayerID) {
	gm.RemovePlayerFromJoinQueue(playerID)
	gm.RemovePlayerFromLeaveQueue(playerID)
	gm.RemovePlayerFromGame(playerID)

	gm.mu.Lock()
	p, exists := gm.allPlayers[playerID]
	if exists {
		delete(gm.allPlayers, playerID)
	}
	gm.mu.Unlock()

	if exists && p != nil {
		p.Close()
		slog.Info("Player unregistered", "playerID", playerID)
	}
}

func (gm *GameManager) RemovePlayerFromJoinQueue(playerID PlayerID) {
	gm.mu.Lock()
	gm.joinQueue = slices.DeleteFunc(gm.joinQueue, func(p PlayerID) bool {
		return p == playerID
	})
	gm.mu.Unlock()
}

func (gm *GameManager) GetGame(gameID GameID) *Game {
	gm.mu.Lock()
	defer gm.mu.Unlock()
	return gm.games[gameID]
}

func (gm *GameManager) JoinPlayer(playerID PlayerID) {
	gm.joinChan <- playerID
}

func (gm *GameManager) LeavePlayer(playerID PlayerID) {
	gm.leaveChan <- playerID
}

func (gm *GameManager) RecvGameCreation() <-chan *Game {
	return gm.gameCreatedChan
}

func (gm *GameManager) sendGameCreated(game *Game) {
	gm.gameCreatedChan <- game
	slog.Info("Sending game created event", "gameID", game.ID)
}

func (gm *GameManager) IsPlayerInJoinQueue(playerID PlayerID) bool {
	gm.mu.Lock()
	defer gm.mu.Unlock()
	return slices.Contains(gm.joinQueue, playerID)
}

func (gm *GameManager) HandleJoinQueue() {
	for {
		select {
		case playerID, ok := <-gm.joinChan:
			if !ok {
				return
			}
			gm.RemovePlayerFromLeaveQueue(playerID)
			gm.enqueueJoin(playerID)
			gm.checkMatchmaking()
		case <-gm.parentCtx.Done():
			slog.Info("HandleJoinQueue loop exiting due to context cancel")
			return
		}
	}
}

func (gm *GameManager) enqueueJoin(playerID PlayerID) {
	gm.mu.Lock()
	defer gm.mu.Unlock()
	for _, id := range gm.joinQueue {
		if id == playerID {
			return
		}
	}
	gm.joinQueue = append(gm.joinQueue, playerID)
	slog.Info("Updated Join queue length", "length", len(gm.joinQueue))
}

func (gm *GameManager) RemovePlayerFromLeaveQueue(playerID PlayerID) {
	gm.mu.Lock()
	gm.leaveQueue = slices.DeleteFunc(gm.leaveQueue, func(p PlayerID) bool {
		return p == playerID
	})
	gm.mu.Unlock()
}

func (gm *GameManager) HandleLeaveQueue() {
	for {
		select {
		case playerID, ok := <-gm.leaveChan:
			if !ok {
				return
			}
			gm.mu.Lock()
			gm.leaveQueue = append(gm.leaveQueue, playerID)
			gm.mu.Unlock()
			slog.Info("Updated leave queue length", "playerID", playerID)
			gm.RemovePlayerFromGame(playerID)
		case <-gm.parentCtx.Done():
			slog.Info("HandleLeaveQueue loop exiting due to context cancel")
			return
		}
	}
}

func (gm *GameManager) checkMatchmaking() {
	for {
		gm.mu.Lock()
		if len(gm.joinQueue) < 2 {
			gm.mu.Unlock()
			return
		}
		player1ID := gm.joinQueue[0]
		player2ID := gm.joinQueue[1]
		gm.joinQueue = gm.joinQueue[2:]
		gm.mu.Unlock()
		go gm.createAndSendGame(player1ID, player2ID)
	}
}

func (gm *GameManager) createAndSendGame(player1ID, player2ID PlayerID) {
	slog.Info("Creating and sending game with players", "player1ID", player1ID, "player2ID", player2ID)

	gm.mu.Lock()
	if slices.Contains(gm.leaveQueue, player1ID) || slices.Contains(gm.leaveQueue, player2ID) {
		gm.mu.Unlock()
		slog.Info("Player left while matchmaking", "player1ID", player1ID, "player2ID", player2ID)
		return
	}

	player1 := gm.allPlayers[player1ID]
	player2 := gm.allPlayers[player2ID]
	if player1 == nil || player2 == nil {
		gm.mu.Unlock()
		slog.Error("Player not found in allPlayers during matchmaking", "player1ID", player1ID, "player2ID", player2ID)
		return
	}

	game := NewGame(gm.parentCtx, player1, player2)
	gm.games[game.ID] = game
	snapshot := game.State.Snapshot()
	result := NewMatchResult(game.ID, snapshot)
	gm.mu.Unlock()

	player1.SendMatchReady(result)
	player2.SendMatchReady(result)
	slog.Info("Game created", "gameID", game.ID, "player1ID", player1ID, "player2ID", player2ID, "event", "matched")
	gm.sendGameCreated(game)
}

func (gm *GameManager) RemoveGame(gameID GameID) {
	gm.mu.Lock()
	game, ok := gm.games[gameID]
	if ok {
		delete(gm.games, gameID)
	}
	gm.mu.Unlock()
	if ok && game != nil {
		game.Cancel()
	}
}

func (gm *GameManager) RemovePlayerFromGame(playerID PlayerID) {
	gm.mu.Lock()
	var gameID GameID
	var winner PlayerID
	for id, g := range gm.games {
		g.mu.Lock()
		_, inGame := g.Players[playerID]
		if inGame {
			gameID = id
			if playerID == g.State.Player1.PlayerID {
				winner = g.State.Player2.PlayerID
			} else {
				winner = g.State.Player1.PlayerID
			}
		}
		g.mu.Unlock()
		if gameID != "" {
			break
		}
	}
	gm.mu.Unlock()

	if gameID != "" {
		gm.endGame(gameID, winner)
	}
}

func (gm *GameManager) endGame(gameID GameID, winner PlayerID) {
	game := gm.GetGame(gameID)
	if game == nil {
		return
	}
	if winner != "" {
		// Session loop broadcasts and removes the game when it receives the winner.
		game.State.SetGameWinner(winner)
		return
	}
	gm.RemoveGame(gameID)
}

func (gm *GameManager) Shutdown() {
	gm.mu.Lock()
	defer gm.mu.Unlock()
	slog.Info("Shutting down GameManager, cancelling active games...")
	for _, g := range gm.games {
		g.Cancel()
	}
}

func (gm *GameManager) Close() {
	close(gm.Register)
	close(gm.Unregister)
	close(gm.joinChan)
	close(gm.leaveChan)
}
