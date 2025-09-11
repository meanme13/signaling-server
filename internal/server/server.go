package server

import (
	"context"
	"fmt"
	"signaling-server/internal/config"
	"signaling-server/internal/logger"
	"signaling-server/internal/metrics"
	"signaling-server/internal/middleware"
	"signaling-server/internal/ws"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

func Run(ctx context.Context) error {
	cfg := config.Load()
	app := fiber.New(fiber.Config{
		AppName: cfg.AppName,
	})

	middleware.Register(app)

	metrics.Register(app, cfg.AppName)

	ws.RegisterRoutes(app)

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	addr := fmt.Sprintf(":%d", cfg.Port)
	logger.Log.Info("server starting", zap.String("addr", addr))

	go func() {
		if err := app.Listen(addr); err != nil {
			logger.Log.Fatal("server stopped", zap.Error(err))
		}
	}()

	<-ctx.Done()
	logger.Log.Info("shutting down server...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := app.ShutdownWithContext(shutdownCtx); err != nil {
		logger.Log.Error("failed to shutdown gracefully", zap.Error(err))
		return err
	}

	logger.Log.Info("server exited gracefully")
	return nil
}
