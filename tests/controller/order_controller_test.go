package controller

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/benbeisheim/crypto-exchange-server/internal/controller"
	"github.com/benbeisheim/crypto-exchange-server/internal/exchange"
	"github.com/benbeisheim/crypto-exchange-server/internal/service"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockOrderService is a mock implementation of the OrderService
type MockOrderService struct {
	mock.Mock
}

func (m *MockOrderService) ExecuteBuy(amount float64, symbol string) (*service.Order, error) {
	args := m.Called(amount, symbol)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.Order), args.Error(1)
}

func (m *MockOrderService) ExecuteSell(amount float64, symbol string) (*service.Order, error) {
	args := m.Called(amount, symbol)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.Order), args.Error(1)
}

// Helper function to create a new mock OrderService that implements the service.OrderService interface
func newMockOrderService() *service.OrderService {
	mockKraken := &exchange.KrakenClient{}
	mockCoinbase := &exchange.CoinbaseClient{}
	return service.NewOrderService(mockKraken, mockCoinbase)
}

func setupTest() (*fiber.App, *MockOrderService) {
	app := fiber.New()
	mockService := new(MockOrderService)
	orderService := newMockOrderService()
	controller := controller.NewOrderController(orderService)

	app.Get("/buy", controller.HandleBuy)
	app.Get("/sell", controller.HandleSell)

	return app, mockService
}

func TestHandleBuy(t *testing.T) {
	tests := []struct {
		name           string
		amount         float64
		symbol         string
		mockResponse   *service.Order
		mockError      error
		expectedStatus int
		expectedBody   string
	}{
		{
			name:   "successful buy",
			amount: 1.0,
			symbol: "BTC-USD",
			mockResponse: &service.Order{
				LowPrice:  30000.0,
				HighPrice: 30100.0,
				AvgPrice:  30050.0,
				Exchanges: []string{"coinbase", "kraken"},
				TotalSize: 1.0,
				Symbol:    "BTC-USD",
			},
			mockError:      nil,
			expectedStatus: 200,
			expectedBody:   `{"lowPrice":30000,"highPrice":30100,"avgPrice":30050,"exchange":["coinbase","kraken"],"totalSize":1,"symbol":"BTC-USD"}`,
		},
		{
			name:           "invalid amount",
			amount:         0,
			symbol:         "BTC-USD",
			mockResponse:   nil,
			mockError:      nil,
			expectedStatus: 400,
			expectedBody:   `{"error":"Invalid amount"}`,
		},
		{
			name:           "missing symbol",
			amount:         1.0,
			symbol:         "",
			mockResponse:   nil,
			mockError:      nil,
			expectedStatus: 400,
			expectedBody:   `{"error":"Symbol is required"}`,
		},
		{
			name:           "invalid symbol format",
			amount:         1.0,
			symbol:         "BTCUSD",
			mockResponse:   nil,
			mockError:      nil,
			expectedStatus: 400,
			expectedBody:   `{"error":"invalid symbol format. Must be in 'BASE-QUOTE' format (e.g., 'BTC-USD')"}`,
		},
		{
			name:           "service error",
			amount:         1.0,
			symbol:         "BTC-USD",
			mockResponse:   nil,
			mockError:      fmt.Errorf("insufficient liquidity"),
			expectedStatus: 500,
			expectedBody:   `{"error":"insufficient liquidity"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, mockService := setupTest()

			if tt.mockResponse != nil || tt.mockError != nil {
				mockService.On("ExecuteBuy", tt.amount, tt.symbol).Return(tt.mockResponse, tt.mockError)
			}

			req := fmt.Sprintf("/buy?amount=%f&symbol=%s", tt.amount, tt.symbol)
			resp, err := app.Test(httptest.NewRequest("GET", req, nil))

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			body := make(map[string]interface{})
			err = json.NewDecoder(resp.Body).Decode(&body)
			assert.NoError(t, err)

			expectedBody := make(map[string]interface{})
			err = json.Unmarshal([]byte(tt.expectedBody), &expectedBody)
			assert.NoError(t, err)

			assert.Equal(t, expectedBody, body)
			mockService.AssertExpectations(t)
		})
	}
}

func TestHandleSell(t *testing.T) {
	tests := []struct {
		name           string
		amount         float64
		symbol         string
		mockResponse   *service.Order
		mockError      error
		expectedStatus int
		expectedBody   string
	}{
		{
			name:   "successful sell",
			amount: 1.0,
			symbol: "BTC-USD",
			mockResponse: &service.Order{
				LowPrice:  30000.0,
				HighPrice: 30100.0,
				AvgPrice:  30050.0,
				Exchanges: []string{"coinbase", "kraken"},
				TotalSize: 1.0,
				Symbol:    "BTC-USD",
			},
			mockError:      nil,
			expectedStatus: 200,
			expectedBody:   `{"lowPrice":30000,"highPrice":30100,"avgPrice":30050,"exchange":["coinbase","kraken"],"totalSize":1,"symbol":"BTC-USD"}`,
		},
		{
			name:           "invalid amount",
			amount:         0,
			symbol:         "BTC-USD",
			mockResponse:   nil,
			mockError:      nil,
			expectedStatus: 400,
			expectedBody:   `{"error":"Invalid amount"}`,
		},
		{
			name:           "missing symbol",
			amount:         1.0,
			symbol:         "",
			mockResponse:   nil,
			mockError:      nil,
			expectedStatus: 400,
			expectedBody:   `{"error":"Symbol is required"}`,
		},
		{
			name:           "invalid symbol format",
			amount:         1.0,
			symbol:         "BTCUSD",
			mockResponse:   nil,
			mockError:      nil,
			expectedStatus: 400,
			expectedBody:   `{"error":"invalid symbol format. Must be in 'BASE-QUOTE' format (e.g., 'BTC-USD')"}`,
		},
		{
			name:           "service error",
			amount:         1.0,
			symbol:         "BTC-USD",
			mockResponse:   nil,
			mockError:      fmt.Errorf("insufficient liquidity"),
			expectedStatus: 500,
			expectedBody:   `{"error":"insufficient liquidity"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, mockService := setupTest()

			if tt.mockResponse != nil || tt.mockError != nil {
				// For sell operations, we need to use the Kraken symbol format (without hyphen)
				krakenSymbol := tt.symbol
				if tt.symbol != "" {
					krakenSymbol = strings.ReplaceAll(tt.symbol, "-", "")
				}
				mockService.On("ExecuteSell", tt.amount, krakenSymbol).Return(tt.mockResponse, tt.mockError)
			}

			req := fmt.Sprintf("/sell?amount=%f&symbol=%s", tt.amount, tt.symbol)
			resp, err := app.Test(httptest.NewRequest("GET", req, nil))

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			body := make(map[string]interface{})
			err = json.NewDecoder(resp.Body).Decode(&body)
			assert.NoError(t, err)

			expectedBody := make(map[string]interface{})
			err = json.Unmarshal([]byte(tt.expectedBody), &expectedBody)
			assert.NoError(t, err)

			assert.Equal(t, expectedBody, body)
			mockService.AssertExpectations(t)
		})
	}
}
