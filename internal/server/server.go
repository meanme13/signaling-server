package server

import (
	"context"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"go.uber.org/zap"

	"signaling-server/internal/config"
	"signaling-server/internal/logger"
	"signaling-server/internal/metrics"
	"signaling-server/internal/middleware"
	"signaling-server/internal/ws/handlers"
	"signaling-server/internal/ws/routes"
)

func Run(ctx context.Context, cfg *config.Config) error {
	app := fiber.New(fiber.Config{
		AppName:      cfg.AppName,
		ReadTimeout:  cfg.FiberReadTimeout,
		WriteTimeout: cfg.FiberWriteTimeout,
		BodyLimit:    cfg.FiberBodyLimit,
	})

	if err := middleware.Register(app, cfg); err != nil {
		logger.Log.Error("failed to register middleware", zap.Error(err))
		return err
	}
	metrics.Register(app, cfg.AppName)
	routes.RegisterPubKeyRoute(app)

	app.Get("/ws", websocket.New(func(c *websocket.Conn) {
		handlers.Handler(cfg, c)
	}))

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	addr := fmt.Sprintf(":%d", cfg.Port)
	logger.Log.Info("server starting", zap.String("addr", addr))

	errCh := make(chan error, 1)

	go func() {
		if err := app.Listen(addr); err != nil {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		logger.Log.Info("received shutdown signal, starting graceful shutdown", zap.String("reason", ctx.Err().Error()))
	case err := <-errCh:
		logger.Log.Error("server failed to start or stopped unexpectedly", zap.Error(err))
		return err
	}

	logger.Log.Info("shutting down server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := app.ShutdownWithContext(shutdownCtx); err != nil {
		logger.Log.Error("failed to shutdown gracefully", zap.Error(err))
		return fmt.Errorf("shutdown error: %w", err)
	}

	logger.Log.Info("server exited gracefully")
	return nil
}
