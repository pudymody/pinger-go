package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/pudymody/pinger-go/endpoint"
	"github.com/pudymody/pinger-go/hit"
	"github.com/pudymody/pinger-go/storage"
	"github.com/pudymody/pinger-go/web"
	"github.com/pudymody/pinger-go/worker"
)

func main() {
	ctx, cancelCtx := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	defer cancelCtx()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	sqlite, err := storage.NewSqlite(ctx, "/data/db.db")
	if err != nil {
		logger.ErrorContext(ctx, "Opening storage", "error", err)
		return
	}

	endpointService := endpoint.NewEndpointService(sqlite)
	hitService := hit.NewHitService(sqlite)

	subPath := "/"
	if subPathENV := os.Getenv("BASE_PATH"); subPathENV != "" {
		subPath = subPathENV
	}

	addr := ":8080"
	if addrENV := os.Getenv("HTTP_ADDR"); addrENV != "" {
		addr = addrENV
	}

	server := web.NewServer(addr, subPath, &endpointService, &hitService, logger.With("name", "server"))
	err = server.Start(ctx)
	if err != nil {
		logger.ErrorContext(ctx, "Opening server", "error", err)
		return
	}

	workerInstance := worker.NewWorker(&endpointService, &hitService, logger.With("name", "worker"))
	err = workerInstance.Start(ctx)
	if err != nil {
		logger.ErrorContext(ctx, "Opening worker", "error", err)
		return
	}

	<-ctx.Done()
	err = server.Shutdown(context.Background())
	if err != nil {
		logger.ErrorContext(ctx, "Shutting down server", "error", err)
	}
	logger.InfoContext(ctx, "Server stopped")

	err = workerInstance.Shutdown(context.Background())
	if err != nil {
		logger.ErrorContext(ctx, "Shutting down worker", "error", err)
	}
	logger.InfoContext(ctx, "Worker stopped")
}
