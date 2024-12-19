// internal/service/order_service.go
package service

import (
	"github.com/benbeisheim/crypto-exchange-server/internal/exchange"
)

type OrderService struct {
	binanceClient  exchange.Client
	coinbaseClient exchange.Client
}

func NewOrderService(binance, coinbase exchange.Client) *OrderService {
	return &OrderService{
		binanceClient:  binance,
		coinbaseClient: coinbase,
	}
}

func (s *OrderService) ExecuteBuy(amount float64, symbol string) (*Order, error) {
	// Get prices from both exchanges
	binancePrice, err := s.binanceClient.GetPrice(symbol)
	if err != nil {
		return nil, err
	}

	coinbasePrice, err := s.coinbaseClient.GetPrice(symbol)
	if err != nil {
		return nil, err
	}

	// Choose best price
	var bestPrice *exchange.ExchangePrice
	if binancePrice.Price <= coinbasePrice.Price {
		bestPrice = binancePrice
	} else {
		bestPrice = coinbasePrice
	}

	return &Order{
		BtcAmount: amount,
		UsdAmount: amount * bestPrice.Price,
		Exchange:  []string{bestPrice.Exchange},
	}, nil
}

func (s *OrderService) ExecuteSell(amount float64, symbol string) (*Order, error) {
	// Similar to ExecuteBuy but look for highest price
	binancePrice, err := s.binanceClient.GetPrice(symbol)
	if err != nil {
		return nil, err
	}

	coinbasePrice, err := s.coinbaseClient.GetPrice(symbol)
	if err != nil {
		return nil, err
	}

	var bestPrice *exchange.ExchangePrice
	if binancePrice.Price >= coinbasePrice.Price {
		bestPrice = binancePrice
	} else {
		bestPrice = coinbasePrice
	}

	return &Order{
		BtcAmount: amount,
		UsdAmount: amount * bestPrice.Price,
		Exchange:  []string{bestPrice.Exchange},
	}, nil
}

type Order struct {
	BtcAmount float64  `json:"btcAmount"`
	UsdAmount float64  `json:"usdAmount"`
	Exchange  []string `json:"exchange"`
}
