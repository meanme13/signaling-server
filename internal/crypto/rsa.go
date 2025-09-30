package crypto

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"sync"

	"go.uber.org/zap"

	"signaling-server/internal/logger"
	"signaling-server/internal/utils"
)

type rsaKeyStore struct {
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
	mu         sync.RWMutex
}

var rsaKeys = &rsaKeyStore{}

func InitRSA(bits int) error {
	if bits < 2048 {
		logger.Log.Error("RSA key size too small", zap.Int("bits", bits))
		return fmt.Errorf("crypto: RSA key size must be at least 2048 bits, got %d", bits)
	}

	if err := utils.CheckRandomReader(); err != nil {
		logger.Log.Error("random reader unavailable", zap.Error(err))
		return fmt.Errorf("crypto: %w", err)
	}

	key, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		logger.Log.Error("failed to generate RSA key", zap.Error(err))
		return fmt.Errorf("crypto: failed to generate RSA key: %w", err)
	}

	rsaKeys.mu.Lock()
	defer rsaKeys.mu.Unlock()
	rsaKeys.privateKey = key
	rsaKeys.publicKey = &key.PublicKey
	return nil
}

func GetPublicKey() (*rsa.PublicKey, error) {
	rsaKeys.mu.RLock()
	defer rsaKeys.mu.RUnlock()
	if rsaKeys.publicKey == nil {
		logger.Log.Error("public key not initialized")
		return nil, fmt.Errorf("crypto: public key not initialized")
	}
	return rsaKeys.publicKey, nil
}

func SerializePublicKey(format string) (string, error) {
	pub, err := GetPublicKey()
	if err != nil {
		logger.Log.Error("failed to get public key", zap.Error(err))
		return "", err
	}

	pubASN1, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		logger.Log.Error("failed to marshal public key", zap.Error(err))
		return "", fmt.Errorf("crypto: failed to marshal public key: %w", err)
	}

	switch format {
	case "pem":
		pubPEM := pem.EncodeToMemory(&pem.Block{
			Type:  "PUBLIC KEY",
			Bytes: pubASN1,
		})
		return string(pubPEM), nil
	case "spki-base64":
		return base64.StdEncoding.EncodeToString(pubASN1), nil
	default:
		logger.Log.Error("unsupported public key format", zap.String("format", format))
		return "", fmt.Errorf("crypto: unsupported public key format: %s", format)
	}
}

func EncryptRSA(msg []byte, pub *rsa.PublicKey) ([]byte, error) {
	if err := utils.CheckRandomReader(); err != nil {
		logger.Log.Error("random reader unavailable", zap.Error(err))
		return nil, fmt.Errorf("crypto: %w", err)
	}

	ciphertext, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, pub, msg, nil)
	if err != nil {
		logger.Log.Error("failed to encrypt RSA", zap.Error(err))
		return nil, fmt.Errorf("crypto: failed to encrypt RSA: %w", err)
	}
	return ciphertext, nil
}

func DecryptRSA(ciphertext []byte) ([]byte, error) {
	rsaKeys.mu.RLock()
	defer rsaKeys.mu.RUnlock()
	if rsaKeys.privateKey == nil {
		logger.Log.Error("private key not initialized")
		return nil, fmt.Errorf("crypto: private key not initialized")
	}

	if err := utils.CheckRandomReader(); err != nil {
		logger.Log.Error("random reader unavailable", zap.Error(err))
		return nil, fmt.Errorf("crypto: %w", err)
	}

	plaintext, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, rsaKeys.privateKey, ciphertext, nil)
	if err != nil {
		logger.Log.Error("failed to decrypt RSA", zap.Error(err))
		return nil, fmt.Errorf("crypto: failed to decrypt RSA: %w", err)
	}
	return plaintext, nil
}
