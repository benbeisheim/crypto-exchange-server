package service

import (
	"errors"
	"testing"

	"github.com/benbeisheim/crypto-exchange-server/internal/exchange"
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
	service := NewOrderService(krakenClient, coinbaseClient)

	assert.NotNil(t, service)
	assert.Equal(t, krakenClient, service.krakenClient)
	assert.Equal(t, coinbaseClient, service.coinbaseClient)
}

// Test invalid order level parsing
func TestInvalidOrderLevelParsing(t *testing.T) {
	service := NewOrderService(nil, nil)

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

// Test order book aggregation with invalid data
func TestAggregateOrderBooksWithInvalidData(t *testing.T) {
	service := NewOrderService(nil, nil)

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

func TestExecuteSell(t *testing.T) {
	tests := []struct {
		name          string
		amount        float64
		symbol        string
		krakenBook    *exchange.OrderBook
		coinbaseBook  *exchange.OrderBook
		expectError   bool
		mockError     error
		expectedOrder *Order
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
			expectError: false,
			expectedOrder: &Order{
				LowPrice:  30002.00,
				HighPrice: 30003.00,
				AvgPrice:  30002.50,
				Exchanges: []string{"kraken", "coinbase"},
				TotalSize: 2.0,
				Symbol:    "BTC-USD",
			},
		},
		{
			name:        "kraken api error",
			amount:      1.0,
			symbol:      "BTC-USD",
			expectError: true,
			mockError:   errors.New("api error"),
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
		{
			name:   "empty order books",
			amount: 1.0,
			symbol: "BTC-USD",
			krakenBook: &exchange.OrderBook{
				Bids: [][2]string{},
			},
			coinbaseBook: &exchange.OrderBook{
				Bids: [][2]string{},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockKraken := new(MockExchangeClient)
			mockCoinbase := new(MockExchangeClient)
			service := NewOrderService(mockKraken, mockCoinbase)

			if !tt.expectError {
				mockKraken.On("GetOrderBook", tt.symbol).Return(tt.krakenBook, nil).Once()
				mockCoinbase.On("GetOrderBook", tt.symbol).Return(tt.coinbaseBook, nil).Once()
			} else if tt.mockError != nil {
				mockKraken.On("GetOrderBook", tt.symbol).Return(nil, tt.mockError).Once()
			} else {
				mockKraken.On("GetOrderBook", tt.symbol).Return(tt.krakenBook, nil).Once()
				mockCoinbase.On("GetOrderBook", tt.symbol).Return(tt.coinbaseBook, nil).Once()
			}

			order, err := service.ExecuteSell(tt.amount, tt.symbol)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, order)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, order)
				assert.InDelta(t, tt.expectedOrder.LowPrice, order.LowPrice, 0.01)
				assert.InDelta(t, tt.expectedOrder.HighPrice, order.HighPrice, 0.01)
				assert.InDelta(t, tt.expectedOrder.AvgPrice, order.AvgPrice, 0.01)
				assert.Equal(t, tt.expectedOrder.Symbol, order.Symbol)
				assert.Equal(t, tt.expectedOrder.TotalSize, order.TotalSize)
				assert.ElementsMatch(t, tt.expectedOrder.Exchanges, order.Exchanges)
			}

			mockKraken.AssertExpectations(t)
			mockCoinbase.AssertExpectations(t)
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
		expectError   bool
		mockError     error
		expectedOrder *Order
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
			expectError: false,
			expectedOrder: &Order{
				LowPrice:  30000.00,
				HighPrice: 30001.00,
				AvgPrice:  30000.50,
				Exchanges: []string{"kraken", "coinbase"},
				TotalSize: 2.0,
				Symbol:    "BTC-USD",
			},
		},
		{
			name:        "kraken api error",
			amount:      1.0,
			symbol:      "BTC-USD",
			expectError: true,
			mockError:   errors.New("api error"),
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
		{
			name:   "empty order books",
			amount: 1.0,
			symbol: "BTC-USD",
			krakenBook: &exchange.OrderBook{
				Asks: [][2]string{},
			},
			coinbaseBook: &exchange.OrderBook{
				Asks: [][2]string{},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockKraken := new(MockExchangeClient)
			mockCoinbase := new(MockExchangeClient)
			service := NewOrderService(mockKraken, mockCoinbase)

			if !tt.expectError {
				mockKraken.On("GetOrderBook", tt.symbol).Return(tt.krakenBook, nil).Once()
				mockCoinbase.On("GetOrderBook", tt.symbol).Return(tt.coinbaseBook, nil).Once()
			} else if tt.mockError != nil {
				mockKraken.On("GetOrderBook", tt.symbol).Return(nil, tt.mockError).Once()
			} else {
				mockKraken.On("GetOrderBook", tt.symbol).Return(tt.krakenBook, nil).Once()
				mockCoinbase.On("GetOrderBook", tt.symbol).Return(tt.coinbaseBook, nil).Once()
			}

			order, err := service.ExecuteBuy(tt.amount, tt.symbol)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, order)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, order)
				assert.InDelta(t, tt.expectedOrder.LowPrice, order.LowPrice, 0.01)
				assert.InDelta(t, tt.expectedOrder.HighPrice, order.HighPrice, 0.01)
				assert.InDelta(t, tt.expectedOrder.AvgPrice, order.AvgPrice, 0.01)
				assert.Equal(t, tt.expectedOrder.Symbol, order.Symbol)
				assert.Equal(t, tt.expectedOrder.TotalSize, order.TotalSize)
				assert.ElementsMatch(t, tt.expectedOrder.Exchanges, order.Exchanges)
			}

			mockKraken.AssertExpectations(t)
			mockCoinbase.AssertExpectations(t)
		})
	}
}
func TestAggregateOrderBooksBids(t *testing.T) {
	service := NewOrderService(nil, nil)

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
