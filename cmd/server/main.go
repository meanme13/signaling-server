package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"signaling-server/internal/config"
	"signaling-server/internal/crypto"
	"signaling-server/internal/logger"
	"signaling-server/internal/redis"
	"signaling-server/internal/server"
	"syscall"

	"go.uber.org/zap"
)

func main() {
	if err := crypto.InitRSA(2048); err != nil {
		log.Fatalf("Failed to init crypto: %v", err)
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	if err := logger.Init(cfg.LogFilePath); err != nil {
		log.Fatalf("Failed to init logger: %v", err)
	}
	defer func() {
		if err := logger.Sync(); err != nil {
			log.Printf("Failed to sync logger: %v", err)
		}
	}()

	if err := redis.Init(&cfg.Redis); err != nil {
		logger.Log.Fatal("Failed to init redis", zap.Error(err))
	}

	if err := redis.InitPubSub(&cfg.Redis); err != nil {
		logger.Log.Fatal("Failed to init Redis PubSub", zap.Error(err))
	}
	defer func() {
		if err := redis.Unsubscribe(); err != nil {
			logger.Log.Error("Failed to unsubscribe from Redis", zap.Error(err))
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := server.Run(ctx, cfg); err != nil {
		logger.Log.Fatal("server failed", zap.Error(err))
	}
}
