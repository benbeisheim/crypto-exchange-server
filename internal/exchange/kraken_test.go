package exchange

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetKrakenOrderBook(t *testing.T) {
	tests := []struct {
		name           string
		symbol         string
		mockResponse   string
		mockStatusCode int
		expectError    bool
		expectedBook   *OrderBook
	}{
		{
			name:   "successful response",
			symbol: "BTC-USD",
			mockResponse: `{
				"error": [],
				"result": {
					"BTC-USD": {
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
			expectedBook: &OrderBook{
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
			expectedBook: &OrderBook{
				Asks:     [][2]string{},
				Bids:     [][2]string{},
				Exchange: "kraken",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify the request URL contains the correct pair parameter
				expectedPair := strings.ReplaceAll(tt.symbol, "-", "")
				if r.URL.Query().Get("pair") != expectedPair {
					t.Errorf("Expected pair %s, got %s", expectedPair, r.URL.Query().Get("pair"))
				}

				// Return mock response
				w.WriteHeader(tt.mockStatusCode)
				io.WriteString(w, tt.mockResponse)
			}))
			defer server.Close()

			// Create a client with the test server URL
			client := &KrakenClient{
				baseURL: server.URL,
			}

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
	client := NewKrakenClient()
	assert.NotNil(t, client)
	assert.IsType(t, &KrakenClient{}, client)
}
