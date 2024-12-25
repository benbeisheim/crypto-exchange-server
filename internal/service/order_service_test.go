package service

import (
	"errors"
	"testing"

	"github.com/benbeisheim/crypto-exchange-server/internal/exchange"
	"github.com/benbeisheim/crypto-exchange-server/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

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

func TestExecuteTransaction(t *testing.T) {
	tests := []struct {
		name          string
		transType     types.TransactionType
		amount        float64
		symbol        string
		krakenBook    *exchange.OrderBook
		coinbaseBook  *exchange.OrderBook
		expectError   bool
		mockError     error
		expectedOrder *Order
	}{
		{
			name:      "successful buy with best price aggregation",
			transType: types.Buy,
			amount:    2.0,
			symbol:    "BTC-USD",
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
			name:      "successful sell with best price aggregation",
			transType: types.Sell,
			amount:    2.0,
			symbol:    "BTC-USD",
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
			transType:   types.Buy,
			amount:      1.0,
			symbol:      "BTC-USD",
			expectError: true,
			mockError:   errors.New("api error"),
		},
		{
			name:      "insufficient liquidity",
			transType: types.Buy,
			amount:    5.0,
			symbol:    "BTC-USD",
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
			name:      "empty order books",
			transType: types.Buy,
			amount:    1.0,
			symbol:    "BTC-USD",
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

			order, err := service.ExecuteTransaction(tt.amount, tt.symbol, tt.transType)

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

func TestAggregateOrderBooks(t *testing.T) {
	tests := []struct {
		name           string
		krakenLevels   [][2]string
		coinbaseLevels [][2]string
		side           types.OrderSide
		expectError    bool
		expectedLevels []exchange.OrderLevel
	}{
		{
			name: "aggregate bids descending order",
			krakenLevels: [][2]string{
				{"30002.00", "1.0"},
				{"30000.00", "2.0"},
			},
			coinbaseLevels: [][2]string{
				{"30003.00", "1.0"},
				{"30001.00", "2.0"},
			},
			side:        types.Bids,
			expectError: false,
			expectedLevels: []exchange.OrderLevel{
				{Price: 30003.00, Size: 1.0, Exchange: "coinbase"},
				{Price: 30002.00, Size: 1.0, Exchange: "kraken"},
				{Price: 30001.00, Size: 2.0, Exchange: "coinbase"},
				{Price: 30000.00, Size: 2.0, Exchange: "kraken"},
			},
		},
		{
			name: "aggregate asks ascending order",
			krakenLevels: [][2]string{
				{"30000.00", "1.0"},
				{"30002.00", "2.0"},
			},
			coinbaseLevels: [][2]string{
				{"30001.00", "1.0"},
				{"30003.00", "2.0"},
			},
			side:        types.Asks,
			expectError: false,
			expectedLevels: []exchange.OrderLevel{
				{Price: 30000.00, Size: 1.0, Exchange: "kraken"},
				{Price: 30001.00, Size: 1.0, Exchange: "coinbase"},
				{Price: 30002.00, Size: 2.0, Exchange: "kraken"},
				{Price: 30003.00, Size: 2.0, Exchange: "coinbase"},
			},
		},
		{
			name:           "empty levels",
			krakenLevels:   [][2]string{},
			coinbaseLevels: [][2]string{},
			side:           types.Bids,
			expectError:    false,
			expectedLevels: []exchange.OrderLevel{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewOrderService(nil, nil)
			levels, err := service.aggregateOrderBooks(tt.krakenLevels, tt.coinbaseLevels, tt.side)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, len(tt.expectedLevels), len(levels))

			for i, expectedLevel := range tt.expectedLevels {
				assert.InDelta(t, expectedLevel.Price, levels[i].Price, 0.01)
				assert.InDelta(t, expectedLevel.Size, levels[i].Size, 0.01)
				assert.Equal(t, expectedLevel.Exchange, levels[i].Exchange)
			}
		})
	}
}

func TestOrderSideConversion(t *testing.T) {
	tests := []struct {
		name         string
		transType    types.TransactionType
		expectedSide types.OrderSide
	}{
		{
			name:         "buy converts to asks",
			transType:    types.Buy,
			expectedSide: types.Asks,
		},
		{
			name:         "sell converts to bids",
			transType:    types.Sell,
			expectedSide: types.Bids,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			side := tt.transType.ToOrderSide()
			assert.Equal(t, tt.expectedSide, side)
		})
	}
}
