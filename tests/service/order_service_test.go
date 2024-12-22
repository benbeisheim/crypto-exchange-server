package service

import (
	"fmt"
	"testing"

	"github.com/benbeisheim/crypto-exchange-server/internal/exchange"
	"github.com/benbeisheim/crypto-exchange-server/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockExchangeClient implements the exchange.Client interface for testing
type MockExchangeClient struct {
	mock.Mock
}

func (m *MockExchangeClient) GetOrderBook(symbol string) (*exchange.OrderBook, error) {
	args := m.Called(symbol)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*exchange.OrderBook), args.Error(1)
}

func TestNewOrderService(t *testing.T) {
	krakenClient := new(MockExchangeClient)
	coinbaseClient := new(MockExchangeClient)
	service := service.NewOrderService(krakenClient, coinbaseClient)

	assert.NotNil(t, service)
	assert.Equal(t, krakenClient, krakenClient)
	assert.Equal(t, coinbaseClient, coinbaseClient)
}

func TestParseOrderLevel(t *testing.T) {
	service := service.NewOrderService(nil, nil)

	tests := []struct {
		name          string
		level         [2]string
		exchange      string
		expectError   bool
		expectedSize  float64
		expectedPrice float64
	}{
		{
			name:          "valid order level",
			level:         [2]string{"100.50", "1.5"},
			exchange:      "kraken",
			expectError:   false,
			expectedPrice: 100.50,
			expectedSize:  1.5,
		},
		{
			name:        "invalid price",
			level:       [2]string{"invalid", "1.5"},
			exchange:    "coinbase",
			expectError: true,
		},
		{
			name:        "invalid size",
			level:       [2]string{"100.50", "invalid"},
			exchange:    "kraken",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := service.ParseOrderLevel(tt.level, tt.exchange)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedPrice, result.Price)
			assert.Equal(t, tt.expectedSize, result.Size)
			assert.Equal(t, tt.exchange, result.Exchange)
		})
	}
}

func TestExecuteBuy(t *testing.T) {
	tests := []struct {
		name          string
		amount        float64
		symbol        string
		krakenBook    *exchange.OrderBook
		coinbaseBook  *exchange.OrderBook
		krakenError   error
		coinbaseError error
		expectError   bool
		expectedOrder *service.Order
	}{
		{
			name:   "successful buy with best price aggregation",
			amount: 2.0,
			symbol: "BTC-USD",
			krakenBook: &exchange.OrderBook{
				Asks: [][2]string{
					{"30000.00", "1.0"},
					{"30002.00", "2.0"},
				},
				Exchange: "kraken",
			},
			coinbaseBook: &exchange.OrderBook{
				Asks: [][2]string{
					{"30001.00", "1.0"},
					{"30003.00", "2.0"},
				},
				Exchange: "coinbase",
			},
			krakenError:   nil,
			coinbaseError: nil,
			expectError:   false,
			expectedOrder: &service.Order{
				LowPrice:  30000.00,
				HighPrice: 30001.00,
				AvgPrice:  30000.50,
				Exchanges: []string{"kraken", "coinbase"},
				TotalSize: 2.0,
				Symbol:    "BTC-USD",
			},
		},
		{
			name:         "kraken error",
			amount:       1.0,
			symbol:       "BTC-USD",
			krakenBook:   nil,
			coinbaseBook: nil,
			krakenError:  fmt.Errorf("kraken api error"),
			expectError:  true,
		},
		{
			name:          "coinbase error",
			amount:        1.0,
			symbol:        "BTC-USD",
			krakenBook:    nil,
			coinbaseBook:  nil,
			coinbaseError: fmt.Errorf("coinbase api error"),
			expectError:   true,
		},
		{
			name:   "insufficient liquidity",
			amount: 5.0,
			symbol: "BTC-USD",
			krakenBook: &exchange.OrderBook{
				Asks: [][2]string{
					{"30000.00", "1.0"},
				},
			},
			coinbaseBook: &exchange.OrderBook{
				Asks: [][2]string{
					{"30001.00", "1.0"},
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			krakenClient := new(MockExchangeClient)
			coinbaseClient := new(MockExchangeClient)
			service := service.NewOrderService(krakenClient, coinbaseClient)

			// Setup mock responses
			krakenClient.On("GetOrderBook", "BTCUSD").Return(tt.krakenBook, tt.krakenError)
			coinbaseClient.On("GetOrderBook", tt.symbol).Return(tt.coinbaseBook, tt.coinbaseError)

			// Execute test
			order, err := service.ExecuteBuy(tt.amount, tt.symbol)

			// Verify results
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, order)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, order)
			assert.InDelta(t, tt.expectedOrder.LowPrice, order.LowPrice, 0.01)
			assert.InDelta(t, tt.expectedOrder.HighPrice, order.HighPrice, 0.01)
			assert.InDelta(t, tt.expectedOrder.AvgPrice, order.AvgPrice, 0.01)
			assert.Equal(t, tt.expectedOrder.TotalSize, order.TotalSize)
			assert.ElementsMatch(t, tt.expectedOrder.Exchanges, order.Exchanges)
			assert.Equal(t, tt.expectedOrder.Symbol, order.Symbol)

			// Verify mock expectations
			krakenClient.AssertExpectations(t)
			coinbaseClient.AssertExpectations(t)
		})
	}
}

func TestExecuteSell(t *testing.T) {
	tests := []struct {
		name          string
		amount        float64
		symbol        string
		krakenBook    *exchange.OrderBook
		coinbaseBook  *exchange.OrderBook
		krakenError   error
		coinbaseError error
		expectError   bool
		expectedOrder *service.Order
	}{
		{
			name:   "successful sell with best price aggregation",
			amount: 2.0,
			symbol: "BTC-USD",
			krakenBook: &exchange.OrderBook{
				Bids: [][2]string{
					{"30002.00", "1.0"},
					{"30000.00", "2.0"},
				},
				Exchange: "kraken",
			},
			coinbaseBook: &exchange.OrderBook{
				Bids: [][2]string{
					{"30003.00", "1.0"},
					{"30001.00", "2.0"},
				},
				Exchange: "coinbase",
			},
			krakenError:   nil,
			coinbaseError: nil,
			expectError:   false,
			expectedOrder: &service.Order{
				LowPrice:  30002.00,
				HighPrice: 30003.00,
				AvgPrice:  30002.50,
				Exchanges: []string{"kraken", "coinbase"},
			},
		},
		{
			name:         "kraken error",
			amount:       1.0,
			symbol:       "BTC-USD",
			krakenBook:   nil,
			coinbaseBook: nil,
			krakenError:  fmt.Errorf("kraken api error"),
			expectError:  true,
		},
		{
			name:          "coinbase error",
			amount:        1.0,
			symbol:        "BTC-USD",
			krakenBook:    nil,
			coinbaseBook:  nil,
			coinbaseError: fmt.Errorf("coinbase api error"),
			expectError:   true,
		},
		{
			name:   "insufficient liquidity",
			amount: 5.0,
			symbol: "BTC-USD",
			krakenBook: &exchange.OrderBook{
				Bids: [][2]string{
					{"30002.00", "1.0"},
				},
			},
			coinbaseBook: &exchange.OrderBook{
				Bids: [][2]string{
					{"30003.00", "1.0"},
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			krakenClient := new(MockExchangeClient)
			coinbaseClient := new(MockExchangeClient)
			service := service.NewOrderService(krakenClient, coinbaseClient)

			// Setup mock responses
			krakenClient.On("GetOrderBook", tt.symbol).Return(tt.krakenBook, tt.krakenError)
			coinbaseClient.On("GetOrderBook", tt.symbol).Return(tt.coinbaseBook, tt.coinbaseError)

			// Execute test
			order, err := service.ExecuteSell(tt.amount, tt.symbol)

			// Verify results
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, order)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, order)
			assert.InDelta(t, tt.expectedOrder.LowPrice, order.LowPrice, 0.01)
			assert.InDelta(t, tt.expectedOrder.HighPrice, order.HighPrice, 0.01)
			assert.InDelta(t, tt.expectedOrder.AvgPrice, order.AvgPrice, 0.01)
			assert.ElementsMatch(t, tt.expectedOrder.Exchanges, order.Exchanges)

			// Verify mock expectations
			krakenClient.AssertExpectations(t)
			coinbaseClient.AssertExpectations(t)
		})
	}
}

func TestAggregateOrderBooksBids(t *testing.T) {
	service := service.NewOrderService(nil, nil)

	tests := []struct {
		name         string
		krakenBook   *exchange.OrderBook
		coinbaseBook *exchange.OrderBook
		expectError  bool
		expectedBids []exchange.OrderLevel
	}{
		{
			name: "successful aggregation",
			krakenBook: &exchange.OrderBook{
				Bids: [][2]string{
					{"30002.00", "1.0"},
					{"30000.00", "2.0"},
				},
			},
			coinbaseBook: &exchange.OrderBook{
				Bids: [][2]string{
					{"30003.00", "1.0"},
					{"30001.00", "2.0"},
				},
			},
			expectError: false,
			expectedBids: []exchange.OrderLevel{
				{Price: 30003.00, Size: 1.0, Exchange: "coinbase"},
				{Price: 30002.00, Size: 1.0, Exchange: "kraken"},
				{Price: 30001.00, Size: 2.0, Exchange: "coinbase"},
				{Price: 30000.00, Size: 2.0, Exchange: "kraken"},
			},
		},
		{
			name: "empty kraken book",
			krakenBook: &exchange.OrderBook{
				Bids: [][2]string{},
			},
			coinbaseBook: &exchange.OrderBook{
				Bids: [][2]string{
					{"30003.00", "1.0"},
				},
			},
			expectError: false,
			expectedBids: []exchange.OrderLevel{
				{Price: 30003.00, Size: 1.0, Exchange: "coinbase"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bids, err := service.AggregateOrderBooksBids(tt.krakenBook, tt.coinbaseBook)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, len(tt.expectedBids), len(bids))

			for i, expectedBid := range tt.expectedBids {
				assert.InDelta(t, expectedBid.Price, bids[i].Price, 0.01)
				assert.InDelta(t, expectedBid.Size, bids[i].Size, 0.01)
				assert.Equal(t, expectedBid.Exchange, bids[i].Exchange)
			}
		})
	}
}

func TestAggregateOrderBooksAsks(t *testing.T) {
	service := service.NewOrderService(nil, nil)

	tests := []struct {
		name         string
		krakenBook   *exchange.OrderBook
		coinbaseBook *exchange.OrderBook
		expectError  bool
		expectedAsks []exchange.OrderLevel
	}{
		{
			name: "successful aggregation",
			krakenBook: &exchange.OrderBook{
				Asks: [][2]string{
					{"30000.00", "1.0"},
					{"30002.00", "2.0"},
				},
			},
			coinbaseBook: &exchange.OrderBook{
				Asks: [][2]string{
					{"30001.00", "1.0"},
					{"30003.00", "2.0"},
				},
			},
			expectError: false,
			expectedAsks: []exchange.OrderLevel{
				{Price: 30000.00, Size: 1.0, Exchange: "kraken"},
				{Price: 30001.00, Size: 1.0, Exchange: "coinbase"},
				{Price: 30002.00, Size: 2.0, Exchange: "kraken"},
				{Price: 30003.00, Size: 2.0, Exchange: "coinbase"},
			},
		},
		{
			name: "empty coinbase book",
			krakenBook: &exchange.OrderBook{
				Asks: [][2]string{
					{"30000.00", "1.0"},
				},
			},
			coinbaseBook: &exchange.OrderBook{
				Asks: [][2]string{},
			},
			expectError: false,
			expectedAsks: []exchange.OrderLevel{
				{Price: 30000.00, Size: 1.0, Exchange: "kraken"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			asks, err := service.AggregateOrderBooksAsks(tt.krakenBook, tt.coinbaseBook)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, len(tt.expectedAsks), len(asks))

			for i, expectedAsk := range tt.expectedAsks {
				assert.InDelta(t, expectedAsk.Price, asks[i].Price, 0.01)
				assert.InDelta(t, expectedAsk.Size, asks[i].Size, 0.01)
				assert.Equal(t, expectedAsk.Exchange, asks[i].Exchange)
			}
		})
	}
}

func TestInvalidOrderLevelParsing(t *testing.T) {
	service := service.NewOrderService(nil, nil)

	tests := []struct {
		name        string
		level       [2]string
		exchange    string
		errorString string
	}{
		{
			name:        "empty price",
			level:       [2]string{"", "1.0"},
			exchange:    "kraken",
			errorString: "error parsing price",
		},
		{
			name:        "empty size",
			level:       [2]string{"30000.00", ""},
			exchange:    "coinbase",
			errorString: "error parsing size",
		},
		{
			name:        "invalid price format",
			level:       [2]string{"30,000.00", "1.0"},
			exchange:    "kraken",
			errorString: "error parsing price",
		},
		{
			name:        "invalid size format",
			level:       [2]string{"30000.00", "1,0"},
			exchange:    "coinbase",
			errorString: "error parsing size",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.ParseOrderLevel(tt.level, tt.exchange)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.errorString)
		})
	}
}

func TestAggregateOrderBooksWithInvalidData(t *testing.T) {
	service := service.NewOrderService(nil, nil)

	tests := []struct {
		name         string
		krakenBook   *exchange.OrderBook
		coinbaseBook *exchange.OrderBook
		testBids     bool
		errorString  string
	}{
		{
			name: "invalid kraken bid price",
			krakenBook: &exchange.OrderBook{
				Bids: [][2]string{{"invalid", "1.0"}},
			},
			coinbaseBook: &exchange.OrderBook{
				Bids: [][2]string{{"30000.00", "1.0"}},
			},
			testBids:    true,
			errorString: "error parsing price",
		},
		{
			name: "invalid coinbase ask size",
			krakenBook: &exchange.OrderBook{
				Asks: [][2]string{{"30000.00", "1.0"}},
			},
			coinbaseBook: &exchange.OrderBook{
				Asks: [][2]string{{"30000.00", "invalid"}},
			},
			testBids:    false,
			errorString: "error parsing size",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error
			if tt.testBids {
				_, err = service.AggregateOrderBooksBids(tt.krakenBook, tt.coinbaseBook)
			} else {
				_, err = service.AggregateOrderBooksAsks(tt.krakenBook, tt.coinbaseBook)
			}
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.errorString)
		})
	}
}

func TestExecutionWithEmptyOrderBooks(t *testing.T) {
	krakenClient := new(MockExchangeClient)
	coinbaseClient := new(MockExchangeClient)
	service := service.NewOrderService(krakenClient, coinbaseClient)

	emptyBook := &exchange.OrderBook{
		Bids: [][2]string{},
		Asks: [][2]string{},
	}

	tests := []struct {
		name        string
		amount      float64
		symbol      string
		operation   string
		errorString string
	}{
		{
			name:        "buy with empty books",
			amount:      1.0,
			symbol:      "BTC-USD",
			operation:   "buy",
			errorString: "no asks available",
		},
		{
			name:        "sell with empty books",
			amount:      1.0,
			symbol:      "BTC-USD",
			operation:   "sell",
			errorString: "no bids available",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			krakenClient.On("GetOrderBook", mock.Anything).Return(emptyBook, nil)
			coinbaseClient.On("GetOrderBook", mock.Anything).Return(emptyBook, nil)

			var err error
			if tt.operation == "buy" {
				_, err = service.ExecuteBuy(tt.amount, tt.symbol)
			} else {
				_, err = service.ExecuteSell(tt.amount, tt.symbol)
			}

			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.errorString)
			krakenClient.AssertExpectations(t)
			coinbaseClient.AssertExpectations(t)
		})
	}
}
