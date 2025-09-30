package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"

	"go.uber.org/zap"
	
	"signaling-server/internal/logger"
	"signaling-server/internal/utils"
)

func GenerateAESKey() ([]byte, error) {
	if err := utils.CheckRandomReader(); err != nil {
		logger.Log.Error("random reader unavailable", zap.Error(err))
		return nil, fmt.Errorf("crypto: %w", err)
	}

	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		logger.Log.Error("failed to generate AES key", zap.Error(err))
		return nil, fmt.Errorf("crypto: failed to generate AES key: %w", err)
	}
	return key, nil
}

func EncryptAES(plaintext, key []byte) ([]byte, error) {
	if err := utils.CheckRandomReader(); err != nil {
		logger.Log.Error("random reader unavailable", zap.Error(err))
		return nil, fmt.Errorf("crypto: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		logger.Log.Error("failed to create AES cipher", zap.Error(err))
		return nil, fmt.Errorf("crypto: failed to create AES cipher: %w", err)
	}

	ciphertext := make([]byte, aes.BlockSize+len(plaintext))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		logger.Log.Error("failed to generate IV", zap.Error(err))
		return nil, fmt.Errorf("crypto: failed to generate IV: %w", err)
	}

	stream := cipher.NewCTR(block, iv)
	stream.XORKeyStream(ciphertext[aes.BlockSize:], plaintext)
	return ciphertext, nil
}

func DecryptAES(ciphertext, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		logger.Log.Error("failed to create AES cipher", zap.Error(err))
		return nil, fmt.Errorf("crypto: failed to create AES cipher: %w", err)
	}

	if len(ciphertext) < aes.BlockSize {
		logger.Log.Error("ciphertext too short")
		return nil, fmt.Errorf("crypto: ciphertext too short")
	}

	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]
	plaintext := make([]byte, len(ciphertext))
	stream := cipher.NewCTR(block, iv)
	stream.XORKeyStream(plaintext, ciphertext)
	return plaintext, nil
}
