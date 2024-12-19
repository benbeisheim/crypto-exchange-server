// internal/exchange/binance.go
package exchange

type BinanceClient struct{}

func NewBinanceClient() *BinanceClient {
	return &BinanceClient{}
}

func (c *BinanceClient) GetPrice(symbol string) (*ExchangePrice, error) {
	// TODO: Implement actual Binance API call
	return &ExchangePrice{
		Price:     40000,
		Available: 1.0,
		Exchange:  "binance",
	}, nil
}
