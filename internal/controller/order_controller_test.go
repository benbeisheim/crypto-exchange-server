package controller

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/benbeisheim/crypto-exchange-server/internal/exchange"
	"github.com/benbeisheim/crypto-exchange-server/internal/service"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockExchangeClient implements the exchange.Client interface
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

func setupTest() (*fiber.App, *MockExchangeClient, *MockExchangeClient) {
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			return c.Status(400).JSON(fiber.Map{
				"error": err.Error(),
			})
		},
	})

	mockKraken := new(MockExchangeClient)
	mockCoinbase := new(MockExchangeClient)

	orderService := service.NewOrderService(mockKraken, mockCoinbase)
	controller := NewOrderController(orderService)

	app.Get("/buy", controller.HandleBuy)
	app.Get("/sell", controller.HandleSell)

	return app, mockKraken, mockCoinbase
}

func TestHandleBuy(t *testing.T) {
	tests := []struct {
		name           string
		amount         float64
		symbol         string
		setupMocks     bool
		mockKraken     *exchange.OrderBook
		mockCoinbase   *exchange.OrderBook
		mockError      error
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name:       "successful buy",
			amount:     2.0,
			symbol:     "BTC-USD",
			setupMocks: true,
			mockKraken: &exchange.OrderBook{
				Asks: [][2]string{
					{"30000.00", "1.0"},
					{"30002.00", "2.0"},
				},
				Exchange: "kraken",
			},
			mockCoinbase: &exchange.OrderBook{
				Asks: [][2]string{
					{"30001.00", "1.0"},
					{"30003.00", "2.0"},
				},
				Exchange: "coinbase",
			},
			mockError:      nil,
			expectedStatus: 200,
			expectedBody: map[string]interface{}{
				"lowPrice":  30000.0,
				"highPrice": 30001.0,
				"avgPrice":  30000.5,
				"exchange":  []interface{}{"kraken", "coinbase"},
				"totalSize": 2.0,
				"symbol":    "BTC-USD",
			},
		},
		{
			name:           "invalid amount",
			amount:         0,
			symbol:         "BTC-USD",
			setupMocks:     false,
			expectedStatus: 400,
			expectedBody: map[string]interface{}{
				"error": "Invalid amount",
			},
		},
		{
			name:           "missing symbol",
			amount:         1.0,
			symbol:         "",
			setupMocks:     false,
			expectedStatus: 400,
			expectedBody: map[string]interface{}{
				"error": "Symbol is required",
			},
		},
		{
			name:           "invalid symbol format",
			amount:         1.0,
			symbol:         "BTCUSD",
			setupMocks:     false,
			expectedStatus: 400,
			expectedBody: map[string]interface{}{
				"error": "invalid symbol format. Must be in 'BASE-QUOTE' format (e.g., 'BTC-USD')",
			},
		},
		{
			name:           "service error",
			amount:         1.0,
			symbol:         "BTC-USD",
			setupMocks:     true,
			mockError:      errors.New("insufficient liquidity"),
			expectedStatus: 500,
			expectedBody: map[string]interface{}{
				"error": "kraken error: insufficient liquidity",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, mockKraken, mockCoinbase := setupTest()

			if tt.setupMocks {
				if tt.mockError != nil {
					// For Kraken, remove hyphen from symbol
					mockKraken.On("GetOrderBook", tt.symbol).Return(nil, tt.mockError).Once()
				} else {
					mockKraken.On("GetOrderBook", tt.symbol).Return(tt.mockKraken, nil).Once()
					mockCoinbase.On("GetOrderBook", tt.symbol).Return(tt.mockCoinbase, nil).Once()
				}
			}

			req := httptest.NewRequest("GET", "/buy?amount="+
				fmt.Sprintf("%f", tt.amount)+"&symbol="+tt.symbol, nil)

			resp, err := app.Test(req)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			// In both TestHandleBuy and TestHandleSell, replace the body decoding and assertion section with this:

			var body map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&body)
			if tt.expectedBody != nil {
				assert.NoError(t, err)

				// Extract exchange arrays for comparison
				actualExchanges, hasExchanges := body["exchange"].([]interface{})
				expectedExchanges, expectedHasExchanges := tt.expectedBody["exchange"].([]interface{})

				if hasExchanges && expectedHasExchanges {
					// Compare exchanges array independently of order
					assert.ElementsMatch(t, expectedExchanges, actualExchanges)

					// Create copies of maps without the exchange field
					bodyWithoutExchanges := make(map[string]interface{})
					expectedWithoutExchanges := make(map[string]interface{})

					for k, v := range body {
						if k != "exchange" {
							bodyWithoutExchanges[k] = v
						}
					}
					for k, v := range tt.expectedBody {
						if k != "exchange" {
							expectedWithoutExchanges[k] = v
						}
					}

					// Compare everything else
					assert.Equal(t, expectedWithoutExchanges, bodyWithoutExchanges)
				} else {
					// For error cases, compare entire response
					assert.Equal(t, tt.expectedBody, body)
				}
			}

			if tt.setupMocks {
				mockKraken.AssertExpectations(t)
				mockCoinbase.AssertExpectations(t)
			}
		})
	}
}

func TestHandleSell(t *testing.T) {
	tests := []struct {
		name           string
		amount         float64
		symbol         string
		setupMocks     bool
		mockKraken     *exchange.OrderBook
		mockCoinbase   *exchange.OrderBook
		mockError      error
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name:       "successful sell",
			amount:     2.0,
			symbol:     "BTC-USD",
			setupMocks: true,
			mockKraken: &exchange.OrderBook{
				Bids: [][2]string{
					{"30002.00", "1.0"},
					{"30000.00", "2.0"},
				},
				Exchange: "kraken",
			},
			mockCoinbase: &exchange.OrderBook{
				Bids: [][2]string{
					{"30003.00", "1.0"},
					{"30001.00", "2.0"},
				},
				Exchange: "coinbase",
			},
			mockError:      nil,
			expectedStatus: 200,
			expectedBody: map[string]interface{}{
				"lowPrice":  30002.0,
				"highPrice": 30003.0,
				"avgPrice":  30002.5,
				"exchange":  []interface{}{"kraken", "coinbase"},
				"totalSize": 2.0,
				"symbol":    "BTC-USD",
			},
		},
		{
			name:           "invalid amount",
			amount:         0,
			symbol:         "BTC-USD",
			setupMocks:     false,
			expectedStatus: 400,
			expectedBody: map[string]interface{}{
				"error": "Invalid amount",
			},
		},
		{
			name:           "missing symbol",
			amount:         1.0,
			symbol:         "",
			setupMocks:     false,
			expectedStatus: 400,
			expectedBody: map[string]interface{}{
				"error": "Symbol is required",
			},
		},
		{
			name:           "invalid symbol format",
			amount:         1.0,
			symbol:         "BTCUSD",
			setupMocks:     false,
			expectedStatus: 400,
			expectedBody: map[string]interface{}{
				"error": "invalid symbol format. Must be in 'BASE-QUOTE' format (e.g., 'BTC-USD')",
			},
		},
		{
			name:           "service error",
			amount:         1.0,
			symbol:         "BTC-USD",
			setupMocks:     true,
			mockError:      errors.New("insufficient liquidity"),
			expectedStatus: 500,
			expectedBody: map[string]interface{}{
				"error": "kraken error: insufficient liquidity",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, mockKraken, mockCoinbase := setupTest()

			if tt.setupMocks {
				if tt.mockError != nil {
					mockKraken.On("GetOrderBook", tt.symbol).Return(nil, tt.mockError).Once()
				} else {
					mockKraken.On("GetOrderBook", tt.symbol).Return(tt.mockKraken, nil).Once()
					mockCoinbase.On("GetOrderBook", tt.symbol).Return(tt.mockCoinbase, nil).Once()
				}
			}

			req := httptest.NewRequest("GET", "/sell?amount="+
				fmt.Sprintf("%f", tt.amount)+"&symbol="+tt.symbol, nil)

			resp, err := app.Test(req)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			// In both TestHandleBuy and TestHandleSell, replace the body decoding and assertion section with this:

			var body map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&body)
			if tt.expectedBody != nil {
				assert.NoError(t, err)

				// Extract exchange arrays for comparison
				actualExchanges, hasExchanges := body["exchange"].([]interface{})
				expectedExchanges, expectedHasExchanges := tt.expectedBody["exchange"].([]interface{})

				if hasExchanges && expectedHasExchanges {
					// Compare exchanges array independently of order
					assert.ElementsMatch(t, expectedExchanges, actualExchanges)

					// Create copies of maps without the exchange field
					bodyWithoutExchanges := make(map[string]interface{})
					expectedWithoutExchanges := make(map[string]interface{})

					for k, v := range body {
						if k != "exchange" {
							bodyWithoutExchanges[k] = v
						}
					}
					for k, v := range tt.expectedBody {
						if k != "exchange" {
							expectedWithoutExchanges[k] = v
						}
					}

					// Compare everything else
					assert.Equal(t, expectedWithoutExchanges, bodyWithoutExchanges)
				} else {
					// For error cases, compare entire response
					assert.Equal(t, tt.expectedBody, body)
				}
			}

			if tt.setupMocks {
				mockKraken.AssertExpectations(t)
				mockCoinbase.AssertExpectations(t)
			}
		})
	}
}
