// internal/controller/order_controller.go
package controller

import (
	"github.com/benbeisheim/crypto-exchange-server/internal/service"
	"github.com/benbeisheim/crypto-exchange-server/internal/util"
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

	// Validate amount
	if amount <= 0 {
		return ctx.Status(400).JSON(fiber.Map{
			"error": "Invalid amount",
		})
	}

	// Validate symbol
	if symbol == "" {
		return ctx.Status(400).JSON(fiber.Map{
			"error": "Symbol is required",
		})
	}

	// Validate symbol format
	if err := util.ValidateSymbol(symbol); err != nil {
		return ctx.Status(400).JSON(fiber.Map{
			"error": err.Error(),
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

	// Validate amount
	if amount <= 0 {
		return ctx.Status(400).JSON(fiber.Map{
			"error": "Invalid amount",
		})
	}

	// Validate symbol
	if symbol == "" {
		return ctx.Status(400).JSON(fiber.Map{
			"error": "Symbol is required",
		})
	}

	// Validate symbol format
	if err := util.ValidateSymbol(symbol); err != nil {
		return ctx.Status(400).JSON(fiber.Map{
			"error": err.Error(),
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
