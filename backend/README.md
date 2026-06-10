# Air Hockey Game Server (Go)

This directory contains the backend for the Air Hockey game, a stable, showcase-ready game server written in Go. It supports HTTP for player matchmaking and session lifecycle management, and UDP for real-time game play.

## Environment Variables

| Variable | Description | Default |
|---|---|---|
| `PORT` | HTTP server port | `8080` |
| `UDPPORT` | UDP server port | `8050` |
| `REDIS_URL` | Redis URL for player identity storage | `redis://localhost:6379` |
| `TICKRATE` | Game physics tick rate (frames per second) | `30` |

## Run Instructions

### Prerequisites
- Go 1.22+ (tested with Go 1.26)
- Redis server running locally or via Docker

### Starting Redis
To run Redis locally via Docker:
```bash
docker run -d --name air-hockey-redis -p 6379:6379 redis:alpine
```

### Running the server
To start the backend HTTP and UDP servers:
```bash
cd backend
go run ./cmd
```

## Running Unit Tests

The test suite contains small, high-signal tests covering the core game logic and HTTP handlers. It does not require a running Redis instance to execute (uses an in-memory mock store).

To run all tests:
```bash
cd backend
go test ./...
```

To run tests with the Go race detector (requires CGO to be enabled):
```bash
cd backend
# On Unix-like systems:
go test -race ./...
# On Windows (requires gcc / CGO_ENABLED=1):
$env:CGO_ENABLED="1"
go test -race ./...
```

## API Summary

### HTTP Endpoints

* `POST /players/resolve`
  * Body: `{ "name": "PlayerName" }`
  * Response: `{ "playerID": "uuid-string", "displayName": "PlayerName" }`
  * Description: Resolves the display name to a player UUID (restores cached UUID or creates a new one). Cased names are preserved.
* `GET /players/{playerID}`
  * Response: `{ "playerID": "uuid-string", "displayName": "PlayerName" }`
  * Description: Checks player registration state.
* `POST /join`
  * Body: `{ "playerID": "uuid-string" }`
  * Description: Enqueues the player into the matchmaking queue.
* `GET /matchStatus?playerID={playerID}`
  * Response: `{ "GameID": "game-uuid", "State": { ... } }` (or `timeout` / `request cancelled`)
  * Description: Long polls until a match is found and returns the initial game state.
* `POST /leave`
  * Body: `{ "playerID": "uuid-string" }`
  * Description: Removes a player from the matchmaking queue.
* `POST /unregister`
  * Body: `{ "playerID": "uuid-string" }`
  * Description: Unregisters the player and forfeits any active games.

### UDP Protocol

Real-time paddle positioning and game state updates happen over UDP (default port `8050`). Clients send periodic coordinate packets, and the server broadcasts state snapshots back to both player UDP addresses.

## Player Identity & Security Note

Display names are for reinstall recovery and in-game messages only. Names are not authenticated or verified via passwords/OAuth. This is a simple, lightweight registration system intended for non-public demos.
