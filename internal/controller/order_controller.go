// internal/controller/order_controller.go
package controller

import (
	"github.com/benbeisheim/crypto-exchange-server/internal/service"
	"github.com/benbeisheim/crypto-exchange-server/internal/types"
	"github.com/benbeisheim/crypto-exchange-server/internal/util"
	"github.com/gofiber/fiber/v2"
)

type OrderController struct {
	service service.OrderServiceInterface
}

func NewOrderController(service service.OrderServiceInterface) *OrderController {
	return &OrderController{
		service: service,
	}
}

func (c *OrderController) validateRequest(ctx *fiber.Ctx) (float64, string, error) {
	amount := ctx.QueryFloat("amount", 0)
	symbol := ctx.Query("symbol")

	if amount <= 0 {
		return 0, "", fiber.NewError(fiber.StatusBadRequest, "Invalid amount")
	}

	if symbol == "" {
		return 0, "", fiber.NewError(fiber.StatusBadRequest, "Symbol is required")
	}

	if err := util.ValidateSymbolFormat(symbol); err != nil {
		return 0, "", fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	return amount, symbol, nil
}

func (c *OrderController) HandleBuy(ctx *fiber.Ctx) error {
	amount, symbol, err := c.validateRequest(ctx)
	if err != nil {
		if fiberErr, ok := err.(*fiber.Error); ok {
			return ctx.Status(fiberErr.Code).JSON(fiber.Map{
				"error": fiberErr.Message,
			})
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	order, err := c.service.ExecuteTransaction(amount, symbol, types.Buy)
	if err != nil {
		return ctx.Status(500).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return ctx.JSON(order)
}

func (c *OrderController) HandleSell(ctx *fiber.Ctx) error {
	amount, symbol, err := c.validateRequest(ctx)
	if err != nil {
		if fiberErr, ok := err.(*fiber.Error); ok {
			return ctx.Status(fiberErr.Code).JSON(fiber.Map{
				"error": fiberErr.Message,
			})
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	order, err := c.service.ExecuteTransaction(amount, symbol, types.Sell)
	if err != nil {
		return ctx.Status(500).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return ctx.JSON(order)
}
