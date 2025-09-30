package config

import "time"

type RedisConfig struct {
	Host          string
	Password      string
	DB            int
	TTL           time.Duration
	PubSubChannel string
}

type Config struct {
	AppName             string
	Port                int
	FiberReadTimeout    time.Duration
	FiberWriteTimeout   time.Duration
	FiberBodyLimit      int
	LogFilePath         string
	MetricsPath         string
	AllowedOrigins      []string
	AllowMethods        []string
	AllowHeaders        []string
	AllowCredentials    bool
	RateLimitMax        int
	RateLimitExpiration time.Duration
	Redis               RedisConfig
	DefaultRoomLimit    int
	RoomKeyPrefix       string
}
