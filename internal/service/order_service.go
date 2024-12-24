// internal/service/order_service.go
package service

import (
	"fmt"
	"math"
	"strconv"

	"github.com/benbeisheim/crypto-exchange-server/internal/exchange"
)

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

// parse OrderLevel price/size into float
func (s *OrderService) ParseOrderLevel(level [2]string, exchangeName string) (exchange.OrderLevel, error) {
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

func (s *OrderService) AggregateOrderBooksBids(krakenBook, coinbaseBook *exchange.OrderBook) ([]exchange.OrderLevel, error) {
	var allBids []exchange.OrderLevel
	krakenIndex := 0
	coinbaseIndex := 0

	for krakenIndex < len(krakenBook.Bids) || coinbaseIndex < len(coinbaseBook.Bids) {
		// If no more kraken bids, append coinbase bid
		if krakenIndex >= len(krakenBook.Bids) {
			orderLevel, err := s.ParseOrderLevel(coinbaseBook.Bids[coinbaseIndex], "coinbase")
			if err != nil {
				return nil, fmt.Errorf("coinbase bid error: %v", err)
			}
			allBids = append(allBids, orderLevel)
			coinbaseIndex++
			continue
		}

		// If no more coinbase bids, append kraken bid
		if coinbaseIndex >= len(coinbaseBook.Bids) {
			orderLevel, err := s.ParseOrderLevel(krakenBook.Bids[krakenIndex], "kraken")
			if err != nil {
				return nil, fmt.Errorf("kraken bid error: %v", err)
			}
			allBids = append(allBids, orderLevel)
			krakenIndex++
			continue
		}

		// Parse both prices for comparison
		krakenOrderLevel, err := s.ParseOrderLevel(krakenBook.Bids[krakenIndex], "kraken")
		if err != nil {
			return nil, fmt.Errorf("kraken bid error: %v", err)
		}

		coinbaseOrderLevel, err := s.ParseOrderLevel(coinbaseBook.Bids[coinbaseIndex], "coinbase")
		if err != nil {
			return nil, fmt.Errorf("coinbase bid error: %v", err)
		}

		// Compare prices and append highest (opposite from asks)
		if krakenOrderLevel.Price > coinbaseOrderLevel.Price {
			allBids = append(allBids, krakenOrderLevel)
			krakenIndex++
		} else {
			allBids = append(allBids, coinbaseOrderLevel)
			coinbaseIndex++
		}
	}

	return allBids, nil
}

func (s *OrderService) AggregateOrderBooksAsks(krakenBook, coinbaseBook *exchange.OrderBook) ([]exchange.OrderLevel, error) {
	var allAsks []exchange.OrderLevel
	krakenIndex := 0
	coinbaseIndex := 0

	for krakenIndex < len(krakenBook.Asks) || coinbaseIndex < len(coinbaseBook.Asks) {
		// If no more kraken asks, append coinbase ask
		if krakenIndex >= len(krakenBook.Asks) {
			orderLevel, err := s.ParseOrderLevel(coinbaseBook.Asks[coinbaseIndex], "coinbase")
			if err != nil {
				return nil, fmt.Errorf("coinbase ask error: %v", err)
			}
			allAsks = append(allAsks, orderLevel)
			coinbaseIndex++
			continue
		}

		// If no more coinbase asks, append kraken ask
		if coinbaseIndex >= len(coinbaseBook.Asks) {
			orderLevel, err := s.ParseOrderLevel(krakenBook.Asks[krakenIndex], "kraken")
			if err != nil {
				return nil, fmt.Errorf("kraken ask error: %v", err)
			}
			allAsks = append(allAsks, orderLevel)
			krakenIndex++
			continue
		}

		// Parse both prices for comparison
		krakenOrderLevel, err := s.ParseOrderLevel(krakenBook.Asks[krakenIndex], "kraken")
		if err != nil {
			return nil, fmt.Errorf("kraken ask error: %v", err)
		}

		coinbaseOrderLevel, err := s.ParseOrderLevel(coinbaseBook.Asks[coinbaseIndex], "coinbase")
		if err != nil {
			return nil, fmt.Errorf("coinbase ask error: %v", err)
		}

		// Compare prices and append lowest
		if krakenOrderLevel.Price < coinbaseOrderLevel.Price {
			allAsks = append(allAsks, krakenOrderLevel)
			krakenIndex++
		} else {
			allAsks = append(allAsks, coinbaseOrderLevel)
			coinbaseIndex++
		}
	}

	return allAsks, nil
}

func (s *OrderService) ExecuteSell(amount float64, symbol string) (*Order, error) {
	// Get order books from both exchanges
	krakenBook, err := s.krakenClient.GetOrderBook(symbol)
	if err != nil {
		return nil, fmt.Errorf("kraken error: %v", err)
	}

	coinbaseBook, err := s.coinbaseClient.GetOrderBook(symbol)
	if err != nil {
		return nil, fmt.Errorf("coinbase error: %v", err)
	}

	// Aggregate bids from both exchanges
	allBids, err := s.AggregateOrderBooksBids(krakenBook, coinbaseBook)
	if err != nil {
		return nil, fmt.Errorf("error aggregating order books: %v", err)
	}

	if len(allBids) == 0 {
		return nil, fmt.Errorf("no bids available for %s", symbol)
	}

	// Initialize tracking variables
	var totalRevenue float64
	var filledAmount float64
	exchanges := make(map[string]bool)
	remainingAmount := amount
	highPrice := allBids[0].Price // First bid is highest price since bids are sorted descending
	var lowPrice float64          // Will be set by last used bid

	// Fill order from aggregated bids
	for _, level := range allBids {
		if remainingAmount <= 0 {
			break
		}

		lowPrice = level.Price // Will end up being lowest price used

		// Calculate fill at this level
		fillAmount := math.Min(remainingAmount, level.Size)
		revenue := fillAmount * level.Price

		// Update totals
		totalRevenue += revenue
		filledAmount += fillAmount
		remainingAmount -= fillAmount
		exchanges[level.Exchange] = true
	}

	// Check if order was completely filled
	if remainingAmount > 0 {
		return nil, fmt.Errorf("insufficient liquidity to fill order of size %f, only filled %f", amount, filledAmount)
	}

	// Convert exchanges map to slice
	var exchangeList []string
	for exchange := range exchanges {
		exchangeList = append(exchangeList, exchange)
	}

	// Create and return order summary
	return &Order{
		LowPrice:  lowPrice,
		HighPrice: highPrice,
		AvgPrice:  totalRevenue / filledAmount,
		Exchanges: exchangeList,
		TotalSize: filledAmount,
		Symbol:    symbol,
	}, nil
}

func (s *OrderService) ExecuteBuy(amount float64, symbol string) (*Order, error) {
	// Get order books from both exchanges
	krakenBook, err := s.krakenClient.GetOrderBook(symbol)
	if err != nil {
		return nil, fmt.Errorf("kraken error: %v", err)
	}

	coinbaseBook, err := s.coinbaseClient.GetOrderBook(symbol)
	if err != nil {
		return nil, fmt.Errorf("coinbase error: %v", err)
	}

	// Aggregate asks from both exchanges
	allAsks, err := s.AggregateOrderBooksAsks(krakenBook, coinbaseBook)
	if err != nil {
		return nil, fmt.Errorf("error aggregating order books: %v", err)
	}

	if len(allAsks) == 0 {
		return nil, fmt.Errorf("no asks available for %s", symbol)
	}

	// Initialize tracking variables
	var totalCost float64
	var filledAmount float64
	exchanges := make(map[string]bool)
	remainingAmount := amount
	lowPrice := allAsks[0].Price // First ask is lowest price since asks are sorted ascending
	var highPrice float64        // Will be set by last used ask

	// Fill order from aggregated asks
	for _, level := range allAsks {
		if remainingAmount <= 0 {
			break
		}

		highPrice = level.Price // Will end up being highest price used

		// Calculate fill at this level
		fillAmount := math.Min(remainingAmount, level.Size)
		cost := fillAmount * level.Price

		// Update totals
		totalCost += cost
		filledAmount += fillAmount
		remainingAmount -= fillAmount
		exchanges[level.Exchange] = true
	}

	// Check if order was completely filled
	if remainingAmount > 0 {
		return nil, fmt.Errorf("insufficient liquidity to fill order of size %f, only filled %f", amount, filledAmount)
	}

	// Convert exchanges map to slice
	var exchangeList []string
	for exchange := range exchanges {
		exchangeList = append(exchangeList, exchange)
	}

	// Create and return order summary
	return &Order{
		LowPrice:  lowPrice,
		HighPrice: highPrice,
		AvgPrice:  totalCost / filledAmount,
		Exchanges: exchangeList,
		TotalSize: filledAmount,
		Symbol:    symbol,
	}, nil
}

type Order struct {
	LowPrice  float64  `json:"lowPrice"`
	HighPrice float64  `json:"highPrice"`
	AvgPrice  float64  `json:"avgPrice"`
	Exchanges []string `json:"exchange"`
	TotalSize float64  `json:"totalSize"`
	Symbol    string   `json:"symbol"`
}
