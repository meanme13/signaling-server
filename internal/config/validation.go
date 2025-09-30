package config

import (
	"fmt"
	"os"
	"path/filepath"
	"signaling-server/internal/logger"
	"signaling-server/internal/utils"

	"go.uber.org/zap"
)

func Validate(cfg *Config) error {
	if err := utils.ValidateNonEmptyString(cfg.AppName, "APP_NAME"); err != nil {
		return fmt.Errorf("config: %w", err)
	}
	if err := utils.ValidateRegexString(cfg.AppName, "APP_NAME", `^[a-zA-Z0-9-]+$`); err != nil {
		return fmt.Errorf("config: %w", err)
	}

	if err := utils.ValidatePort(cfg.Port, "port"); err != nil {
		return fmt.Errorf("config: %w", err)
	}

	if err := utils.ValidatePositiveInt(cfg.FiberBodyLimit, "fiber body limit"); err != nil {
		return fmt.Errorf("config: %w", err)
	}

	if cfg.LogFilePath != "" {
		dir := filepath.Dir(cfg.LogFilePath)
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			if err := os.MkdirAll(dir, 0755); err != nil {
				logger.Log.Error("failed to create log file directory", zap.String("dir", dir), zap.Error(err))
				return fmt.Errorf("config: failed to create log file directory %s: %w", dir, err)
			}
		}
	}

	if cfg.MetricsPath == "" || cfg.MetricsPath[0] != '/' {
		logger.Log.Error("invalid metrics path", zap.String("metrics_path", cfg.MetricsPath))
		return fmt.Errorf("config: invalid metrics path: %s, must be non-empty and start with /", cfg.MetricsPath)
	}

	if len(cfg.AllowedOrigins) == 0 {
		logger.Log.Error("no allowed origins specified")
		return fmt.Errorf("config: no allowed origins specified")
	}
	for _, origin := range cfg.AllowedOrigins {
		if err := utils.ValidateNonEmptyString(origin, "allowed_origins"); err != nil {
			return fmt.Errorf("config: %w", err)
		}
	}

	if len(cfg.AllowMethods) == 0 {
		logger.Log.Error("no allowed methods specified")
		return fmt.Errorf("config: no allowed methods specified")
	}
	for _, method := range cfg.AllowMethods {
		if err := utils.ValidateNonEmptyString(method, "allow_methods"); err != nil {
			return fmt.Errorf("config: %w", err)
		}
	}

	if len(cfg.AllowHeaders) == 0 {
		logger.Log.Error("no allowed headers specified")
		return fmt.Errorf("config: no allowed headers specified")
	}
	for _, header := range cfg.AllowHeaders {
		if err := utils.ValidateNonEmptyString(header, "allowed_headers"); err != nil {
			return fmt.Errorf("config: %w", err)
		}
	}

	if err := utils.ValidatePositiveInt(cfg.RateLimitMax, "rate limit max"); err != nil {
		return fmt.Errorf("config: %w", err)
	}

	if cfg.RateLimitExpiration <= 0 {
		logger.Log.Error("invalid rate limit expiration", zap.Duration("rate_limit_expiration", cfg.RateLimitExpiration))
		return fmt.Errorf("config: invalid rate limit expiration: %v, must be positive", cfg.RateLimitExpiration)
	}

	if err := utils.ValidateNonEmptyString(cfg.Redis.Host, "Redis host"); err != nil {
		return fmt.Errorf("config: %w", err)
	}
	if cfg.Redis.TTL < 0 {
		logger.Log.Error("invalid Redis TTL", zap.Duration("ttl", cfg.Redis.TTL))
		return fmt.Errorf("config: invalid Redis TTL: %v, must be non-negative", cfg.Redis.TTL)
	}
	if err := utils.ValidateNonEmptyString(cfg.Redis.PubSubChannel, "Redis pubsub channel"); err != nil {
		return fmt.Errorf("config: %w", err)
	}

	if err := utils.ValidatePositiveInt(cfg.DefaultRoomLimit, "default room limit"); err != nil {
		return fmt.Errorf("config: %w", err)
	}

	if err := utils.ValidateNonEmptyString(cfg.RoomKeyPrefix, "room key prefix"); err != nil {
		return fmt.Errorf("config: %w", err)
	}

	return nil
}
