package main

import (
	"context"
	"os"
	"os/signal"
	"signaling-server/internal/logger"
	"signaling-server/internal/redis"
	"signaling-server/internal/server"
	"syscall"

	"go.uber.org/zap"
)

func main() {
	logger.Init()
	defer logger.Sync()

	redis.Init("localhost:6379", "", 0)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := server.Run(ctx); err != nil {
		logger.Log.Fatal("server failed", zap.Error(err))
	}
}
