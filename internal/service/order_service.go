// internal/service/order_service.go
package service

import (
	"fmt"
	"math"
	"strconv"

	"github.com/benbeisheim/crypto-exchange-server/internal/exchange"
	"github.com/benbeisheim/crypto-exchange-server/internal/types"
)

type Order struct {
	LowPrice  float64  `json:"lowPrice"`
	HighPrice float64  `json:"highPrice"`
	AvgPrice  float64  `json:"avgPrice"`
	Exchanges []string `json:"exchange"`
	TotalSize float64  `json:"totalSize"`
	Symbol    string   `json:"symbol"`
}

type OrderServiceInterface interface {
	ExecuteTransaction(amount float64, symbol string, transType types.TransactionType) (*Order, error)
}

type OrderService struct {
	krakenClient   exchange.Client
	coinbaseClient exchange.Client
}

func NewOrderService(kraken, coinbase exchange.Client) *OrderService {
	return &OrderService{
		krakenClient:   kraken,
		coinbaseClient: coinbase,
	}
}

func (s *OrderService) parseOrderLevel(level [2]string, exchangeName string) (exchange.OrderLevel, error) {
	// Parse the price and size to float from the level
	price, err := strconv.ParseFloat(level[0], 64)
	if err != nil {
		return exchange.OrderLevel{}, fmt.Errorf("error parsing price: %v", err)
	}

	size, err := strconv.ParseFloat(level[1], 64)
	if err != nil {
		return exchange.OrderLevel{}, fmt.Errorf("error parsing size: %v", err)
	}

	return exchange.OrderLevel{
		Price:    price,
		Size:     size,
		Exchange: exchangeName,
	}, nil
}

func (s *OrderService) aggregateOrderBooks(krakenLevels, coinbaseLevels [][2]string, side types.OrderSide) ([]exchange.OrderLevel, error) {
	var allLevels []exchange.OrderLevel
	krakenIndex := 0
	coinbaseIndex := 0

	isDescending := side == types.Bids

	for krakenIndex < len(krakenLevels) || coinbaseIndex < len(coinbaseLevels) {
		// Handle case where we've exhausted one exchange's orders
		if krakenIndex >= len(krakenLevels) {
			level, err := s.parseOrderLevel(coinbaseLevels[coinbaseIndex], "coinbase")
			if err != nil {
				return nil, err
			}
			allLevels = append(allLevels, level)
			coinbaseIndex++
			continue
		}

		if coinbaseIndex >= len(coinbaseLevels) {
			level, err := s.parseOrderLevel(krakenLevels[krakenIndex], "kraken")
			if err != nil {
				return nil, err
			}
			allLevels = append(allLevels, level)
			krakenIndex++
			continue
		}

		// Parse both levels for comparison
		krakenLevel, err := s.parseOrderLevel(krakenLevels[krakenIndex], "kraken")
		if err != nil {
			return nil, err
		}

		coinbaseLevel, err := s.parseOrderLevel(coinbaseLevels[coinbaseIndex], "coinbase")
		if err != nil {
			return nil, err
		}

		// Compare prices based on order side
		var shouldUseKraken bool
		if isDescending {
			shouldUseKraken = krakenLevel.Price > coinbaseLevel.Price
		} else {
			shouldUseKraken = krakenLevel.Price < coinbaseLevel.Price
		}

		if shouldUseKraken {
			allLevels = append(allLevels, krakenLevel)
			krakenIndex++
		} else {
			allLevels = append(allLevels, coinbaseLevel)
			coinbaseIndex++
		}
	}

	return allLevels, nil
}

func (s *OrderService) ExecuteTransaction(amount float64, symbol string, transType types.TransactionType) (*Order, error) {
	// Get order books from both exchanges
	krakenBook, err := s.krakenClient.GetOrderBook(symbol)
	if err != nil {
		return nil, fmt.Errorf("kraken error: %v", err)
	}

	coinbaseBook, err := s.coinbaseClient.GetOrderBook(symbol)
	if err != nil {
		return nil, fmt.Errorf("coinbase error: %v", err)
	}

	// Determine which side of the order book to use
	side := transType.ToOrderSide()

	// Get the appropriate order levels based on transaction type
	var krakenLevels, coinbaseLevels [][2]string
	if side == types.Bids {
		krakenLevels = krakenBook.Bids
		coinbaseLevels = coinbaseBook.Bids
	} else {
		krakenLevels = krakenBook.Asks
		coinbaseLevels = coinbaseBook.Asks
	}

	// Aggregate orders from both exchanges
	orders, err := s.aggregateOrderBooks(krakenLevels, coinbaseLevels, side)
	if err != nil {
		return nil, fmt.Errorf("error aggregating order books: %v", err)
	}

	if len(orders) == 0 {
		return nil, fmt.Errorf("no %s available for %s", side, symbol)
	}

	// Initialize tracking variables
	var totalCost float64
	var filledAmount float64
	exchanges := make(map[string]bool)
	remainingAmount := amount

	// Track price ranges based on transaction type
	var lowPrice, highPrice float64
	if side == types.Asks {
		lowPrice = orders[0].Price // First ask is lowest for buys
	} else {
		highPrice = orders[0].Price // First bid is highest for sells
	}

	// Fill order from aggregated levels
	for _, level := range orders {
		if remainingAmount <= 0 {
			break
		}

		if side == types.Asks {
			highPrice = level.Price
		} else {
			lowPrice = level.Price
		}

		fillAmount := math.Min(remainingAmount, level.Size)
		cost := fillAmount * level.Price

		totalCost += cost
		filledAmount += fillAmount
		remainingAmount -= fillAmount
		exchanges[level.Exchange] = true
	}

	// Check if order was completely filled
	if remainingAmount > 0 {
		return nil, fmt.Errorf("insufficient liquidity to fill order of size %f, only filled %f",
			amount, filledAmount)
	}

	// Convert exchanges map to slice
	var exchangeList []string
	for exchange := range exchanges {
		exchangeList = append(exchangeList, exchange)
	}

	return &Order{
		LowPrice:  lowPrice,
		HighPrice: highPrice,
		AvgPrice:  totalCost / filledAmount,
		Exchanges: exchangeList,
		TotalSize: filledAmount,
		Symbol:    symbol,
	}, nil
}
