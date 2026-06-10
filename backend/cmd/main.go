package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/Nitin-Poojary/air-hockey/backend/internal/game"
	"github.com/Nitin-Poojary/air-hockey/backend/internal/store"
	"github.com/Nitin-Poojary/air-hockey/backend/internal/utils"
)

func main() {
	// Configure structured slog JSON logger
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	redisURL := utils.GetEnv("REDIS_URL", "redis://localhost:6379")
	redisStore, err := store.NewRedisStore(redisURL)
	if err != nil {
		slog.Error("Failed to initialize Redis store", "error", err)
		os.Exit(1)
	}
	defer redisStore.Close()

	rootCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup

	gameManager := game.NewGameManagerWithContext(rootCtx)
	defer gameManager.Close()

	wg.Go(func() {
		gameManager.Start()
	})
	wg.Go(func() {
		gameManager.HandleJoinQueue()
	})
	wg.Go(func() {
		gameManager.HandleLeaveQueue()
	})

	udpServer, udpConn := startUDPServer(gameManager)
	defer udpConn.Close()

	wg.Go(func() {
		udpServer.StartUDPSession(udpConn)
	})
	wg.Go(func() {
		udpServer.StartTickingOnGameCreation(rootCtx, udpConn)
	})

	mux := setUpRoutes(gameManager, redisStore)
	httpServer := startHTTPServer(mux, utils.GetEnvInt("PORT", "8080"))

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	slog.Info("Server is running. Press Ctrl+C to shut down gracefully.")
	sig := <-sigChan
	slog.Info("Shutdown signal received", "signal", sig.String())

	// 1. Shutdown HTTP server first (stops accepting new connections)
	shutdownCtx, httpCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer httpCancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		slog.Error("HTTP server shutdown error", "error", err)
	} else {
		slog.Info("HTTP server shut down gracefully")
	}

	// 2. Cancel root context to stop loop execution
	cancel()

	// 3. Close UDP connection to unblock ReadFromUDP
	udpConn.Close()

	// 4. Shutdown gameManager (cancels any active game sessions)
	gameManager.Shutdown()

	// 5. Wait for background loops with a timeout
	waitChan := make(chan struct{})
	go func() {
		wg.Wait()
		close(waitChan)
	}()

	select {
	case <-waitChan:
		slog.Info("All background loops exited. Clean shutdown complete.")
	case <-time.After(5 * time.Second):
		slog.Warn("Shutdown timed out waiting for goroutines to exit.")
	}
}
