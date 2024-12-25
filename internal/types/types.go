package types

type TransactionType string
type OrderSide string

const (
	Buy  TransactionType = "buy"
	Sell TransactionType = "sell"
)

const (
	Bids OrderSide = "bids"
	Asks OrderSide = "asks"
)

func (t TransactionType) ToOrderSide() OrderSide {
	switch t {
	case Buy:
		return Asks // When buying, we look at ask prices
	case Sell:
		return Bids // When selling, we look at bid prices
	default:
		panic("invalid transaction type")
	}
}
