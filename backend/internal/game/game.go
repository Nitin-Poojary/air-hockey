package game

import (
	"context"
	"net"
	"sync"
	"time"

	"github.com/google/uuid"
)

type GameID string

type Game struct {
	ID              GameID
	State           *GameState
	Physics         *GamePhysics
	Players         map[PlayerID]*Player
	PlayerAddresses map[PlayerID]*net.UDPAddr
	AddressesMu     sync.Mutex
	mu              sync.Mutex

	ctx             context.Context
	cancel          context.CancelFunc
	roundClearMu    sync.Mutex
	roundClearTimer *time.Timer
}

func NewGame(parent context.Context, player1, player2 *Player) *Game {
	gameState := NewGameState(player1.PlayerID, player2.PlayerID)
	ctx, cancel := context.WithCancel(parent)
	return &Game{
		ID:              generateGameID(),
		State:           gameState,
		Physics:         NewGamePhysics(gameState),
		Players:         map[PlayerID]*Player{player1.PlayerID: player1, player2.PlayerID: player2},
		PlayerAddresses: make(map[PlayerID]*net.UDPAddr),
		ctx:             ctx,
		cancel:          cancel,
	}
}

func (g *Game) Context() context.Context {
	return g.ctx
}

func (g *Game) Cancel() {
	if g.cancel != nil {
		g.cancel()
	}
	g.stopRoundClearTimer()
}

func (g *Game) ScheduleRoundClear(delay time.Duration) {
	g.roundClearMu.Lock()
	defer g.roundClearMu.Unlock()
	if g.roundClearTimer != nil {
		g.roundClearTimer.Stop()
	}
	g.roundClearTimer = time.AfterFunc(delay, func() {
		select {
		case <-g.ctx.Done():
			return
		default:
			g.State.ClearRoundEnded()
		}
	})
}

func (g *Game) stopRoundClearTimer() {
	g.roundClearMu.Lock()
	defer g.roundClearMu.Unlock()
	if g.roundClearTimer != nil {
		g.roundClearTimer.Stop()
		g.roundClearTimer = nil
	}
}

func (g *Game) MovePaddle(playerID PlayerID, x, y float64) bool {
	g.mu.Lock()
	_, ok := g.Players[playerID]
	g.mu.Unlock()
	if !ok {
		return false
	}
	g.State.MovePaddle(playerID, x, y)
	return true
}

func (g *Game) GetPlayer(playerID PlayerID) *Player {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.Players[playerID]
}

func (g *Game) SetPlayerAddress(playerID PlayerID, addr *net.UDPAddr) {
	g.AddressesMu.Lock()
	defer g.AddressesMu.Unlock()
	if g.PlayerAddresses[playerID] == nil {
		g.PlayerAddresses[playerID] = addr
	}
}

func (g *Game) ForEachAddress(fn func(*net.UDPAddr)) {
	g.AddressesMu.Lock()
	defer g.AddressesMu.Unlock()
	for _, addr := range g.PlayerAddresses {
		if addr != nil {
			fn(addr)
		}
	}
}

func generateGameID() GameID {
	return GameID(uuid.New().String())
}
