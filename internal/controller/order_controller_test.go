package controller

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/benbeisheim/crypto-exchange-server/internal/service"
	"github.com/benbeisheim/crypto-exchange-server/internal/types"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/valyala/fasthttp"
)

type MockOrderService struct {
	mock.Mock
}

func (m *MockOrderService) ExecuteTransaction(amount float64, symbol string, transType types.TransactionType) (*service.Order, error) {
	args := m.Called(amount, symbol, transType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.Order), args.Error(1)
}

func setupTest() (*fiber.App, *MockOrderService) {
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			return c.Status(code).JSON(fiber.Map{
				"error": err.Error(),
			})
		},
	})

	mockService := new(MockOrderService)
	controller := NewOrderController(mockService)

	app.Get("/buy", controller.HandleBuy)
	app.Get("/sell", controller.HandleSell)

	return app, mockService
}

func TestHandleRequests(t *testing.T) {
	tests := []struct {
		name           string
		endpoint       string // explicitly specify endpoint
		amount         float64
		symbol         string
		setupMock      bool
		mockResponse   *service.Order
		mockError      error
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name:           "invalid amount buy",
			endpoint:       "/buy",
			amount:         0,
			symbol:         "BTC-USD",
			setupMock:      false, // Important: no mock setup for validation errors
			expectedStatus: 400,
			expectedBody: map[string]interface{}{
				"error": "Invalid amount",
			},
		},
		{
			name:           "missing symbol buy",
			endpoint:       "/buy",
			amount:         1.0,
			symbol:         "",
			setupMock:      false,
			expectedStatus: 400,
			expectedBody: map[string]interface{}{
				"error": "Symbol is required",
			},
		},
		{
			name:      "successful buy",
			endpoint:  "/buy",
			amount:    1.0,
			symbol:    "BTC-USD",
			setupMock: true,
			mockResponse: &service.Order{
				LowPrice:  30000.0,
				HighPrice: 30100.0,
				AvgPrice:  30050.0,
				Exchanges: []string{"kraken", "coinbase"},
				TotalSize: 1.0,
				Symbol:    "BTC-USD",
			},
			expectedStatus: 200,
			expectedBody: map[string]interface{}{
				"lowPrice":  30000.0,
				"highPrice": 30100.0,
				"avgPrice":  30050.0,
				"exchange":  []interface{}{"kraken", "coinbase"},
				"totalSize": 1.0,
				"symbol":    "BTC-USD",
			},
		},
		// Similar cases for sell...
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, mockService := setupTest()

			if tt.setupMock {
				mockService.On("ExecuteTransaction", tt.amount, tt.symbol, types.TransactionType(tt.endpoint[1:])).
					Return(tt.mockResponse, tt.mockError).Once()
			}

			req := httptest.NewRequest("GET", fmt.Sprintf("%s?amount=%f&symbol=%s",
				tt.endpoint, tt.amount, tt.symbol), nil)

			resp, err := app.Test(req)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			var body map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&body)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedBody, body)

			if tt.setupMock {
				mockService.AssertExpectations(t)
			}
		})
	}
}

func TestValidateRequest(t *testing.T) {
	tests := []struct {
		name          string
		amount        float64
		symbol        string
		expectError   bool
		expectedError string
	}{
		{
			name:        "valid request",
			amount:      1.0,
			symbol:      "BTC-USD",
			expectError: false,
		},
		{
			name:          "zero amount",
			amount:        0,
			symbol:        "BTC-USD",
			expectError:   true,
			expectedError: "Invalid amount",
		},
		{
			name:          "negative amount",
			amount:        -1.0,
			symbol:        "BTC-USD",
			expectError:   true,
			expectedError: "Invalid amount",
		},
		{
			name:          "empty symbol",
			amount:        1.0,
			symbol:        "",
			expectError:   true,
			expectedError: "Symbol is required",
		},
		{
			name:          "invalid symbol format",
			amount:        1.0,
			symbol:        "BTCUSD",
			expectError:   true,
			expectedError: "invalid symbol format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, _ := setupTest()
			controller := &OrderController{}

			// Create a new fiber context for testing
			req := httptest.NewRequest("GET", fmt.Sprintf("/test?amount=%f&symbol=%s",
				tt.amount, tt.symbol), nil)
			ctx := app.AcquireCtx(&fasthttp.RequestCtx{})
			ctx.Request().SetRequestURI(req.URL.String())
			defer app.ReleaseCtx(ctx)

			amount, symbol, err := controller.validateRequest(ctx)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.amount, amount)
				assert.Equal(t, tt.symbol, symbol)
			}
		})
	}
}
