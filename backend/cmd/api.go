package main

import (
	"fmt"
	"log/slog"
	"net"
	"net/http"

	"github.com/Nitin-Poojary/air-hockey/backend/internal/game"
	"github.com/Nitin-Poojary/air-hockey/backend/internal/handlers"
	"github.com/Nitin-Poojary/air-hockey/backend/internal/store"
)

func setUpRoutes(gameManager *game.GameManager, redisStore store.PlayerStore) *http.ServeMux {
	handler := handlers.NewHandler(gameManager, redisStore)
	mux := http.NewServeMux()

	mux.HandleFunc("POST /players/resolve", handler.HandleResolve)
	mux.HandleFunc("GET /players/{playerID}", handler.HandleGetPlayer)
	mux.HandleFunc("POST /join", handler.HandleJoin)
	mux.HandleFunc("GET /matchStatus", handler.HandleMatchStatus)
	mux.HandleFunc("POST /leave", handler.HandleLeave)
	mux.HandleFunc("POST /unregister", handler.HandleUnregister)
	return mux
}

func startHTTPServer(mux *http.ServeMux, port int) *http.Server {
	srv := &http.Server{
		Addr:    fmt.Sprintf("0.0.0.0:%d", port),
		Handler: mux,
	}

	go func() {
		slog.Info("HTTP server starting", "port", port)
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			slog.Error("HTTP server ListenAndServe error", "error", err)
		}
	}()

	return srv
}

func startUDPServer(gameManager *game.GameManager) (*handlers.UDPServer, *net.UDPConn) {
	udpServer := handlers.NewUDPServer(gameManager)
	udpConn := udpServer.StartUDPServer()
	return udpServer, udpConn
}
