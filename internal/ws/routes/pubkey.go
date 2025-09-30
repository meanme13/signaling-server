package routes

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"

	"signaling-server/internal/crypto"
	"signaling-server/internal/logger"
)

func RegisterPubKeyRoute(app *fiber.App) {
	app.Get("/pubkey", func(c *fiber.Ctx) error {
		pubPEM, err := crypto.SerializePublicKey("pem")
		if err != nil {
			logger.Log.Error("failed to serialize public key to PEM", zap.Error(err))
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": fmt.Sprintf("failed to serialize public key: %v", err),
			})
		}

		pubBase64, err := crypto.SerializePublicKey("spki-base64")
		if err != nil {
			logger.Log.Error("failed to serialize public key to Base64", zap.Error(err))
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": fmt.Sprintf("failed to serialize public key: %v", err),
			})
		}

		return c.JSON(fiber.Map{
			"pem":     pubPEM,
			"base64":  pubBase64,
			"keyType": "RSA",
		})
	})
}
