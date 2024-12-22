// cmd/server/main.go

package main

import (
	"log"

	"github.com/benbeisheim/crypto-exchange-server/internal/controller"
	"github.com/benbeisheim/crypto-exchange-server/internal/exchange"
	"github.com/benbeisheim/crypto-exchange-server/internal/service"
	"github.com/gofiber/fiber/v2"
)

func main() {
	// Create new Fiber app
	app := fiber.New()

	// Initialize exchange clients
	krakenClient := exchange.NewKrakenClient()
	coinbaseClient := exchange.NewCoinbaseClient()

	// Initialize service with exchange clients
	orderService := service.NewOrderService(krakenClient, coinbaseClient)

	// Initialize controller with service
	orderController := controller.NewOrderController(orderService)

	// Setup routes
	app.Get("/buy", orderController.HandleBuy)
	app.Get("/sell", orderController.HandleSell)

	// Start server
	log.Fatal(app.Listen(":4000"))
}
