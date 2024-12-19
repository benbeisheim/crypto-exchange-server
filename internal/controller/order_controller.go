// internal/controller/order_controller.go
package controller

import (
	"github.com/benbeisheim/crypto-exchange-server/internal/service"
	"github.com/gofiber/fiber/v2"
)

type OrderController struct {
	service *service.OrderService
}

func NewOrderController(service *service.OrderService) *OrderController {
	return &OrderController{
		service: service,
	}
}

func (c *OrderController) HandleBuy(ctx *fiber.Ctx) error {
	amount := ctx.QueryFloat("amount", 0)
	symbol := ctx.Query("symbol")

	if amount <= 0 {
		return ctx.Status(400).JSON(fiber.Map{
			"error": "Invalid amount",
		})
	}

	if symbol == "" {
		return ctx.Status(400).JSON(fiber.Map{
			"error": "Symbol is required",
		})
	}

	order, err := c.service.ExecuteBuy(amount, symbol)
	if err != nil {
		return ctx.Status(500).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return ctx.JSON(order)
}

func (c *OrderController) HandleSell(ctx *fiber.Ctx) error {
	amount := ctx.QueryFloat("amount", 0)
	symbol := ctx.Query("symbol")

	if amount <= 0 {
		return ctx.Status(400).JSON(fiber.Map{
			"error": "Invalid amount",
		})
	}

	if symbol == "" {
		return ctx.Status(400).JSON(fiber.Map{
			"error": "Symbol is required",
		})
	}

	order, err := c.service.ExecuteSell(amount, symbol)
	if err != nil {
		return ctx.Status(500).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return ctx.JSON(order)
}
