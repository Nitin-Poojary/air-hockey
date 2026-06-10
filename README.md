# Air Hockey

A real-time Air Hockey experience built with a Go backend and a Flutter frontend.

## Project Overview

This repository contains:

- `backend/`: Go server handling player registration, matchmaking, Redis-backed identity storage, and UDP game state delivery.
- `frontend/`: Flutter game client using Flame, Bloc, Dio, and UDP to connect to the backend.
- `docker-compose.yml`: Easy development orchestration for Redis and the Go API server.

## Architecture

- HTTP API handles player identity, matchmaking, and session lifecycle.
- UDP transports live paddle movements and game snapshots for real-time gameplay.
- Redis stores player identities and session metadata.
- Flutter frontend renders the game and communicates with the server using both HTTP and UDP.

## Quick Start

### Recommended: Docker Compose

1. From the repository root:

```bash
docker compose up --build
```

2. The backend API will be available at `http://localhost:8080`.
3. Run the Flutter client from `frontend/` after the backend is running.

### Native Local Setup

#### Backend

1. Install Go 1.22+.
2. Start Redis locally or with Docker:

```bash
docker run -d --name air-hockey-redis -p 6379:6379 redis:alpine
```

3. Start the backend server:

```bash
cd backend
go run ./cmd
```

#### Frontend

1. Install Flutter and ensure the SDK is on your PATH.
2. From the frontend folder:

```bash
cd frontend
flutter pub get
```

3. Configure environment settings:
   - Copy `.env.example` to `.env`
   - Update `HTTP_BASE_URL`, `UDP_HOST`, and `UDP_PORT`
4. Run the app:

```bash
flutter run
```

## Frontend Environment

The frontend uses `.env` values to connect to the backend and UDP server.
Copy `frontend/.env.example` to `frontend/.env` and set:

- `HTTP_BASE_URL`
- `UDP_HOST`
- `UDP_PORT`

## Running Tests

### Backend

```bash
cd backend
go test ./...
```

## Useful Commands

- `docker compose up --build`
- `cd backend && go run ./cmd`
- `cd frontend && flutter run`
- `cd backend && go test ./...`

## Notes

- The backend exposes HTTP on port `8080` and UDP on port `8050` by default.
- The frontend may require platform-specific configuration for Android emulators.
