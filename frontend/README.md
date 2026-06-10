# Air Hockey Frontend

Flutter game client for the Air Hockey project.

This frontend uses Flame, Bloc, Dio, and shared preferences to deliver a polished real-time game experience while communicating with the Go backend over HTTP and UDP.

## Project Structure

- `lib/`: Flutter application code.
- `lib/game/`: Game logic and Flame integration.
- `lib/api/`: HTTP client and API services.
- `lib/network/`: UDP transport and real-time connection logic.
- `.env.example`: Example environment settings for backend and UDP host configuration.

## Prerequisites

- Flutter SDK
- Dart SDK (bundled with Flutter)
- Platform tools for your target device/emulator

## Setup

1. Open a terminal in `frontend/`:

```bash
cd frontend
flutter pub get
```

2. Copy the environment template:

```bash
copy .env.example .env
```

3. Update `.env` with your backend and UDP settings:

- `HTTP_BASE_URL`: backend HTTP API endpoint
- `UDP_HOST`: backend UDP server host
- `UDP_PORT`: backend UDP port

## Running the App

From the `frontend/` folder:

```bash
flutter run
```

## Development Notes

- The app is designed to work with the backend API in `backend/`.
- Game state is forwarded across UDP for low-latency paddle control and gameplay updates.
- Player sessions and matchmaking happen through backend HTTP endpoints.

## Helpful Commands

- `flutter pub get`
- `flutter run`
- `flutter doctor`

## Tips

- Ensure the backend server is running before launching the frontend.
- Keep the `.env` file private; it is ignored by Git by default.
