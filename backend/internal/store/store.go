package store

import "context"

type PlayerStore interface {
	GetPlayerIDByName(ctx context.Context, normalized string) (string, error)
	GetPlayerNameByID(ctx context.Context, playerID string) (string, error)
	RegisterPlayerName(ctx context.Context, normalized string, playerID string, originalName string) (bool, error)
}
