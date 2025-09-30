package config

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

func Load() (*Config, error) {
	viper.SetConfigFile(filepath.Join("internal", "config", "server.yaml"))
	viper.AutomaticEnv()

	viper.SetDefault("app_name", "SignalingServer")
	viper.SetDefault("port", 8080)
	viper.SetDefault("fiber_read_timeout", "10s")
	viper.SetDefault("fiber_write_timeout", "10s")
	viper.SetDefault("fiber_body_limit", 4*1024*1024)
	viper.SetDefault("log_file_path", "")
	viper.SetDefault("metrics_path", "/metrics")
	viper.SetDefault("allowed_origins", []string{"http://localhost:3000"})
	viper.SetDefault("allow_methods", []string{"GET", "POST", "HEAD", "OPTIONS"})
	viper.SetDefault("allowed_headers", []string{"Origin", "Content-Type", "Accept"})
	viper.SetDefault("allow_credentials", true)
	viper.SetDefault("rate_limit_max", 100)
	viper.SetDefault("rate_limit_expiration", "1m")
	viper.SetDefault("redis.host", "localhost:6379")
	viper.SetDefault("redis.password", "")
	viper.SetDefault("redis.db", 0)
	viper.SetDefault("redis.ttl", "24h")
	viper.SetDefault("redis.pubsub_channel", "room_events")
	viper.SetDefault("default_room_limit", 2)
	viper.SetDefault("room_key_prefix", "room:")

	if err := viper.ReadInConfig(); err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if !errors.As(err, &configFileNotFoundError) {
			return nil, fmt.Errorf("config: failed to read config file: %w", err)
		}
	}

	cfg := &Config{
		AppName:             viper.GetString("app_name"),
		Port:                viper.GetInt("port"),
		FiberReadTimeout:    viper.GetDuration("fiber_read_timeout"),
		FiberWriteTimeout:   viper.GetDuration("fiber_write_timeout"),
		FiberBodyLimit:      viper.GetInt("fiber_body_limit"),
		LogFilePath:         viper.GetString("log_file_path"),
		MetricsPath:         viper.GetString("metrics_path"),
		AllowedOrigins:      viper.GetStringSlice("allowed_origins"),
		AllowMethods:        viper.GetStringSlice("allow_methods"),
		AllowHeaders:        viper.GetStringSlice("allowed_headers"),
		AllowCredentials:    viper.GetBool("allow_credentials"),
		RateLimitMax:        viper.GetInt("rate_limit_max"),
		RateLimitExpiration: viper.GetDuration("rate_limit_expiration"),
		Redis: RedisConfig{
			Host:          viper.GetString("redis.host"),
			Password:      viper.GetString("redis.password"),
			DB:            viper.GetInt("redis.db"),
			TTL:           viper.GetDuration("redis.ttl"),
			PubSubChannel: viper.GetString("redis.pubsub_channel"),
		},
		DefaultRoomLimit: viper.GetInt("default_room_limit"),
		RoomKeyPrefix:    viper.GetString("room_key_prefix"),
	}

	if err := Validate(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) AllowedStringSlice(param string) (string, error) {
	var result []string
	switch param {
	case "origins":
		result = c.AllowedOrigins
	case "methods":
		result = c.AllowMethods
	case "headers":
		result = c.AllowHeaders
	default:
		return "", fmt.Errorf("config: unknown parameter for AllowedStringSlice: %s", param)
	}
	if len(result) == 0 {
		return "", fmt.Errorf("config: %s list is empty", param)
	}
	return strings.Join(result, ","), nil
}
