package logger

import (
	"fmt"
	"log"
	"strings"

	"go.uber.org/zap"
)

var Log *zap.Logger

func Init(logFilePath string) error {
	cfg := zap.NewProductionConfig()

	outputs := []string{"stderr"}
	if logFilePath != "" {
		outputs = append(outputs, logFilePath)
	}
	cfg.OutputPaths = outputs

	logger, err := cfg.Build()
	if err != nil {
		log.Printf("logger: failed to initialize: %v", err)
		return fmt.Errorf("logger: failed to initialize: %w", err)
	}

	Log = logger
	return nil
}

func Sync() error {
	if Log == nil {
		return nil
	}
	if err := Log.Sync(); err != nil && !strings.Contains(err.Error(), "inappropriate ioctl for device") {
		log.Printf("logger: failed to sync: %v", err)
		return fmt.Errorf("logger: failed to sync: %w", err)
	}
	return nil
}
