package utils

import (
	"crypto/rand"
	"fmt"
	"regexp"

	"go.uber.org/zap"

	"signaling-server/internal/logger"
)

func CheckRandomReader() error {
	if _, err := rand.Read(make([]byte, 1)); err != nil {
		return fmt.Errorf("random reader unavailable: %w", err)
	}
	return nil
}

func ValidateNonEmptyString(value, fieldName string) error {
	if value == "" {
		logger.Log.Error(fmt.Sprintf("%s is empty", fieldName))
		return fmt.Errorf("%s cannot be empty", fieldName)
	}
	return nil
}

func ValidateRegexString(value, fieldName, pattern string) error {
	if !regexp.MustCompile(pattern).MatchString(value) {
		logger.Log.Error(fmt.Sprintf("%s contains invalid characters", fieldName), zap.String(fieldName, value))
		return fmt.Errorf("%s contains invalid characters: %s", fieldName, value)
	}
	return nil
}

func ValidatePositiveInt(value int, fieldName string) error {
	if value <= 0 {
		logger.Log.Error(fmt.Sprintf("invalid %s", fieldName), zap.Int(fieldName, value))
		return fmt.Errorf("invalid %s: %d, must be positive", fieldName, value)
	}
	return nil
}

func ValidatePort(port int, fieldName string) error {
	if port < 1 || port > 65535 {
		logger.Log.Error(fmt.Sprintf("invalid %s", fieldName), zap.Int(fieldName, port))
		return fmt.Errorf("invalid %s: %d, must be between 1 and 65535", fieldName, port)
	}
	return nil
}

func IfThen(cond bool, a, b string) string {
	if cond {
		return a
	}
	return b
}
