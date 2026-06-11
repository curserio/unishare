package main

import (
	"errors"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/curserio/unishare/internal/config"
	"github.com/curserio/unishare/internal/httpapp"
	"github.com/curserio/unishare/internal/store"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("invalid configuration", "err", err)
		os.Exit(1)
	}
	if len(cfg.Users) == 0 {
		slog.Error("UNISHARE_USERS or UNISHARE_TOKEN is required")
		os.Exit(1)
	}

	itemStore, err := store.NewFileStore(cfg.DataDir, cfg.MaxUploadBytes)
	if err != nil {
		slog.Error("failed to initialize store", "err", err)
		os.Exit(1)
	}

	server := &http.Server{
		Addr:              cfg.Addr,
		Handler:           httpapp.New(cfg, itemStore).Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	slog.Info("unishare listening", "addr", cfg.Addr)
	if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		slog.Error("server failed", "err", err)
		os.Exit(1)
	}
}
