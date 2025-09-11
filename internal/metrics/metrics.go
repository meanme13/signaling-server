package metrics

import (
	"github.com/ansrivas/fiberprometheus/v2"
	"github.com/gofiber/fiber/v2"
)

func Register(app *fiber.App, appName string) {
	prometheus := fiberprometheus.New(appName)

	prometheus.RegisterAt(app, "/metrics")

	app.Use(prometheus.Middleware)
}
