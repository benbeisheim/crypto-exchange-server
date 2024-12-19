// internal/exchange/client.go
package exchange

type Client interface {
	GetPrice(symbol string) (*ExchangePrice, error)
}

type ExchangePrice struct {
	Price     float64
	Available float64
	Exchange  string
}
