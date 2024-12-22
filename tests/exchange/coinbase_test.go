package exchange

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/benbeisheim/crypto-exchange-server/internal/exchange"
	"github.com/stretchr/testify/assert"
)

func TestGetOrderBook(t *testing.T) {
	tests := []struct {
		name           string
		symbol         string
		mockResponse   string
		mockStatusCode int
		expectError    bool
		expectedBook   *exchange.OrderBook
	}{
		{
			name:   "successful response",
			symbol: "BTC-USD",
			mockResponse: `{
				"bids": [["30000.00", "1.25", "abc123"], ["29999.00", "0.75", "def456"]],
				"asks": [["30001.00", "1.00", "ghi789"], ["30002.00", "2.00", "jkl012"]]
			}`,
			mockStatusCode: http.StatusOK,
			expectError:    false,
			expectedBook: &exchange.OrderBook{
				Bids: [][2]string{
					{"30000.00", "1.25"},
					{"29999.00", "0.75"},
				},
				Asks: [][2]string{
					{"30001.00", "1.00"},
					{"30002.00", "2.00"},
				},
				Exchange: "coinbase",
			},
		},
		{
			name:           "http error",
			symbol:         "BTC-USD",
			mockResponse:   `{"message": "Internal server error"}`,
			mockStatusCode: http.StatusInternalServerError,
			expectError:    true,
			expectedBook:   nil,
		},
		{
			name:           "invalid json response",
			symbol:         "BTC-USD",
			mockResponse:   `{"bids": [invalid json]}`,
			mockStatusCode: http.StatusOK,
			expectError:    true,
			expectedBook:   nil,
		},
		{
			name:   "empty order book",
			symbol: "BTC-USD",
			mockResponse: `{
				"bids": [],
				"asks": []
			}`,
			mockStatusCode: http.StatusOK,
			expectError:    false,
			expectedBook: &exchange.OrderBook{
				Bids:     [][2]string{},
				Asks:     [][2]string{},
				Exchange: "coinbase",
			},
		},
		{
			name:           "invalid symbol",
			symbol:         "",
			mockResponse:   `{"message": "Invalid symbol"}`,
			mockStatusCode: http.StatusBadRequest,
			expectError:    true,
			expectedBook:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify the request URL contains the symbol
				expectedPath := "/products/" + tt.symbol + "/book"
				if !strings.HasSuffix(r.URL.Path, expectedPath) {
					t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
				}

				// Verify query parameter
				if r.URL.Query().Get("level") != "2" {
					t.Error("Expected level=2 query parameter")
				}

				// Return mock response
				w.WriteHeader(tt.mockStatusCode)
				io.WriteString(w, tt.mockResponse)
			}))
			defer server.Close()

			// Create a client with the test server URL
			client := &exchange.CoinbaseClient{}
			// Override the default Coinbase API URL with our test server URL
			originalURL := "https://api.exchange.coinbase.com"
			defer func() {
				// Reset the URL after the test
				if tt.mockStatusCode == http.StatusOK {
					http.DefaultClient.Get(originalURL)
				}
			}()

			// Execute the test
			book, err := client.GetOrderBook(tt.symbol)

			// Verify results
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, book)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, book)
				assert.Equal(t, tt.expectedBook.Exchange, book.Exchange)
				assert.Equal(t, len(tt.expectedBook.Bids), len(book.Bids))
				assert.Equal(t, len(tt.expectedBook.Asks), len(book.Asks))

				// Compare bid prices and sizes
				for i := range tt.expectedBook.Bids {
					assert.Equal(t, tt.expectedBook.Bids[i][0], book.Bids[i][0], "Bid price mismatch at index %d", i)
					assert.Equal(t, tt.expectedBook.Bids[i][1], book.Bids[i][1], "Bid size mismatch at index %d", i)
				}

				// Compare ask prices and sizes
				for i := range tt.expectedBook.Asks {
					assert.Equal(t, tt.expectedBook.Asks[i][0], book.Asks[i][0], "Ask price mismatch at index %d", i)
					assert.Equal(t, tt.expectedBook.Asks[i][1], book.Asks[i][1], "Ask size mismatch at index %d", i)
				}
			}
		})
	}
}

func TestNewCoinbaseClient(t *testing.T) {
	client := exchange.NewCoinbaseClient()
	assert.NotNil(t, client)
	assert.IsType(t, &exchange.CoinbaseClient{}, client)
}
