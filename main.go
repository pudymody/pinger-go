package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/pudymody/pinger-go/endpoint"
	"github.com/pudymody/pinger-go/hit"
	"github.com/pudymody/pinger-go/storage"
	"github.com/pudymody/pinger-go/web"
)

func main() {
	ctx := context.Background()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	sqlite, err := storage.NewSqlite(ctx, "db.db")
	if err != nil {
		logger.ErrorContext(ctx, "Opening storage", "error", err)
		return
	}

	endpointService := endpoint.NewEndpointService(sqlite)
	hitService := hit.NewHitService(sqlite)

	server := web.NewServer(":8080", "", &endpointService, &hitService, logger)
	server.Start(ctx)

	logger.InfoContext(ctx, "Server stopped")
}
