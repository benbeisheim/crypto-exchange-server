// internal/exchange/coinbase.go
package exchange

type CoinbaseClient struct{}

func NewCoinbaseClient() *CoinbaseClient {
	return &CoinbaseClient{}
}

func (c *CoinbaseClient) GetPrice(symbol string) (*ExchangePrice, error) {
	// TODO: Implement actual Coinbase API call
	return &ExchangePrice{
		Price:     40100,
		Available: 1.0,
		Exchange:  "coinbase",
	}, nil
}
