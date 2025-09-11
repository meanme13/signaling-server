package config

import (
	"log"
	"os"
	"strconv"
)

type Config struct {
	AppName string
	Port    int
}

func Load() *Config {
	portStr := getEnv("PORT", "8080")
	port, err := strconv.Atoi(portStr)
	if err != nil {
		log.Fatalf("invalid PORT: %v", err)
	}

	return &Config{
		AppName: getEnv("APP_NAME", "SignalingServer"),
		Port:    port,
	}
}

func getEnv(key, defaultVal string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return defaultVal
}
