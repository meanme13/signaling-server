package ws

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
)

func RegisterRoutes(app *fiber.App) {
	app.Get("/ws", websocket.New(Handler))
}
