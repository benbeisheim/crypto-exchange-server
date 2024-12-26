package util

import (
	"strings"

	"github.com/gofiber/fiber/v2"
)

// validateSymbolFormat checks if the symbol is in the correct (BASE-QUOTE) format
func ValidateSymbolFormat(symbol string) error {
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
