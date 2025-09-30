package middleware

import (
	"fmt"
	"signaling-server/internal/config"
	"signaling-server/internal/logger"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"go.uber.org/zap"
)

func Register(app *fiber.App, cfg *config.Config) error {
	if len(cfg.AllowedOrigins) == 0 {
		logger.Log.Error("no allowed origins specified for CORS")
		return fmt.Errorf("middleware: no allowed origins specified for CORS")
	}

	logger.Log.Info("registering CORS middleware", zap.Strings("allowed_origins", cfg.AllowedOrigins))

	origins, _ := cfg.AllowedStringSlice("origins")
	if origins == "" {
		logger.Log.Error("CORS AllowOrigins is empty")
		return fmt.Errorf("middleware: CORS AllowOrigins cannot be empty")
	}
	methods, _ := cfg.AllowedStringSlice("methods")
	if methods == "" {
		logger.Log.Error("CORS AllowMethods is empty")
		return fmt.Errorf("middleware: CORS AllowMethods cannot be empty")
	}
	headers, _ := cfg.AllowedStringSlice("headers")
	if headers == "" {
		logger.Log.Error("CORS AllowHeaders is empty")
		return fmt.Errorf("middleware: CORS AllowHeaders cannot be empty")
	}

	app.Use(cors.New(cors.Config{
		AllowOrigins:     origins,
		AllowMethods:     methods,
		AllowHeaders:     headers,
		AllowCredentials: cfg.AllowCredentials,
	}))

	if cfg.RateLimitMax <= 0 {
		logger.Log.Error("invalid rate limit max", zap.Int("max", cfg.RateLimitMax))
		return fmt.Errorf("middleware: invalid rate limit max: %d, must be positive", cfg.RateLimitMax)
	}
	if cfg.RateLimitExpiration <= 0 {
		logger.Log.Error("invalid rate limit expiration", zap.Duration("expiration", cfg.RateLimitExpiration))
		return fmt.Errorf("middleware: invalid rate limit expiration: %v, must be positive", cfg.RateLimitExpiration)
	}

	logger.Log.Info("registering rate limiter middleware",
		zap.Int("max_requests", cfg.RateLimitMax),
		zap.Duration("expiration", cfg.RateLimitExpiration))
	app.Use(limiter.New(limiter.Config{
		Max:        cfg.RateLimitMax,
		Expiration: cfg.RateLimitExpiration,
	}))

	return nil
}
