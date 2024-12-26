package exchange

type Client interface {
	GetOrderBook(symbol string) (*OrderBook, error)
}

type OrderBook struct {
	Bids     [][2]string `json:"bids"`
	Asks     [][2]string `json:"asks"`
	Exchange string
}

type OrderLevel struct {
	Price    float64
	Size     float64
	Exchange string
}
