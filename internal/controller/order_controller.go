// internal/controller/order_controller.go
package controller

import (
	"strings"

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

// validateSymbol checks if the symbol is in the correct format (BASE-QUOTE)
func validateSymbol(symbol string) error {
	// Check if the symbol contains exactly one hyphen
	parts := strings.Split(symbol, "-")
	if len(parts) != 2 {
		return fiber.NewError(400, "invalid symbol format. Must be in 'BASE-QUOTE' format (e.g., 'BTC-USD')")
	}

	// Check that base and quote symbols are not empty
	if len(parts[0]) == 0 || len(parts[1]) == 0 {
		return fiber.NewError(400, "base and quote symbols cannot be empty")
	}

	return nil
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
	if err := validateSymbol(symbol); err != nil {
		return err
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
	if err := validateSymbol(symbol); err != nil {
		return err
	}

	// For kraken, remove the hyphen
	krakenSymbol := strings.ReplaceAll(symbol, "-", "")

	order, err := c.service.ExecuteSell(amount, krakenSymbol)
	if err != nil {
		return ctx.Status(500).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return ctx.JSON(order)
}
