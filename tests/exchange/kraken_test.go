package exchange

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/benbeisheim/crypto-exchange-server/internal/exchange"
	"github.com/stretchr/testify/assert"
)

func TestGetKrakenOrderBook(t *testing.T) {
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
			symbol: "BTCUSD",
			mockResponse: `{
				"error": [],
				"result": {
					"XXBTZUSD": {
						"asks": [
							["30001.00", "1.000", 1623456789],
							["30002.00", "2.000", 1623456790]
						],
						"bids": [
							["30000.00", "1.250", 1623456788],
							["29999.00", "0.750", 1623456787]
						]
					}
				}
			}`,
			mockStatusCode: http.StatusOK,
			expectError:    false,
			expectedBook: &exchange.OrderBook{
				Asks: [][2]string{
					{"30001.00", "1.000"},
					{"30002.00", "2.000"},
				},
				Bids: [][2]string{
					{"30000.00", "1.250"},
					{"29999.00", "0.750"},
				},
				Exchange: "kraken",
			},
		},
		{
			name:   "kraken error response",
			symbol: "BTCUSD",
			mockResponse: `{
				"error": ["EGeneral:Invalid arguments"],
				"result": {}
			}`,
			mockStatusCode: http.StatusOK,
			expectError:    true,
			expectedBook:   nil,
		},
		{
			name:           "http error",
			symbol:         "BTCUSD",
			mockResponse:   `{"error": "Internal server error"}`,
			mockStatusCode: http.StatusInternalServerError,
			expectError:    true,
			expectedBook:   nil,
		},
		{
			name:           "invalid json response",
			symbol:         "BTCUSD",
			mockResponse:   `{"result": {invalid json}}`,
			mockStatusCode: http.StatusOK,
			expectError:    true,
			expectedBook:   nil,
		},
		{
			name:   "empty order book",
			symbol: "BTCUSD",
			mockResponse: `{
				"error": [],
				"result": {
					"XXBTZUSD": {
						"asks": [],
						"bids": []
					}
				}
			}`,
			mockStatusCode: http.StatusOK,
			expectError:    false,
			expectedBook: &exchange.OrderBook{
				Asks:     [][2]string{},
				Bids:     [][2]string{},
				Exchange: "kraken",
			},
		},
		{
			name:   "missing result key",
			symbol: "BTCUSD",
			mockResponse: `{
				"error": [],
				"wrong_key": {}
			}`,
			mockStatusCode: http.StatusOK,
			expectError:    true,
			expectedBook:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify the request path
				assert.Equal(t, "/0/public/Depth", r.URL.Path)

				// Verify the query parameters
				assert.Equal(t, tt.symbol, r.URL.Query().Get("pair"))

				// Verify headers
				assert.Equal(t, "Mozilla/5.0", r.Header.Get("User-Agent"))
				assert.Equal(t, "application/json", r.Header.Get("Accept"))

				// Return mock response
				w.WriteHeader(tt.mockStatusCode)
				io.WriteString(w, tt.mockResponse)
			}))
			defer server.Close()

			// Create a client that uses the test server URL
			client := &exchange.KrakenClient{}

			// Save and replace the default client for testing
			originalHTTPClient := http.DefaultClient
			http.DefaultClient = &http.Client{
				Transport: &http.Transport{
					Proxy: http.ProxyURL(nil),
				},
			}
			defer func() { http.DefaultClient = originalHTTPClient }()

			// Execute the test
			book, err := client.GetOrderBook(tt.symbol)

			// Verify results
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, book)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, book)
			assert.Equal(t, tt.expectedBook.Exchange, book.Exchange)
			assert.Equal(t, len(tt.expectedBook.Asks), len(book.Asks))
			assert.Equal(t, len(tt.expectedBook.Bids), len(book.Bids))

			// Compare asks
			for i, expectedAsk := range tt.expectedBook.Asks {
				assert.Equal(t, expectedAsk[0], book.Asks[i][0], "Ask price mismatch at index %d", i)
				assert.Equal(t, expectedAsk[1], book.Asks[i][1], "Ask size mismatch at index %d", i)
			}

			// Compare bids
			for i, expectedBid := range tt.expectedBook.Bids {
				assert.Equal(t, expectedBid[0], book.Bids[i][0], "Bid price mismatch at index %d", i)
				assert.Equal(t, expectedBid[1], book.Bids[i][1], "Bid size mismatch at index %d", i)
			}
		})
	}
}

func TestNewKrakenClient(t *testing.T) {
	client := exchange.NewKrakenClient()
	assert.NotNil(t, client)
	assert.IsType(t, &exchange.KrakenClient{}, client)
}

func TestKrakenURLFormatting(t *testing.T) {
	tests := []struct {
		name           string
		symbol         string
		expectedPath   string
		expectedSymbol string
	}{
		{
			name:           "standard symbol",
			symbol:         "BTCUSD",
			expectedPath:   "/0/public/Depth",
			expectedSymbol: "BTCUSD",
		},
		{
			name:           "symbol with dash",
			symbol:         "BTC-USD",
			expectedPath:   "/0/public/Depth",
			expectedSymbol: "BTC-USD",
		},
		{
			name:           "lowercase symbol",
			symbol:         "btcusd",
			expectedPath:   "/0/public/Depth",
			expectedSymbol: "btcusd",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, tt.expectedPath, r.URL.Path)
				assert.Equal(t, tt.expectedSymbol, r.URL.Query().Get("pair"))
				w.WriteHeader(http.StatusOK)
				io.WriteString(w, `{"error":[],"result":{"XXBTZUSD":{"asks":[],"bids":[]}}}`)
			}))
			defer server.Close()

			client := &exchange.KrakenClient{}
			_, err := client.GetOrderBook(tt.symbol)
			assert.NoError(t, err)
		})
	}
}
