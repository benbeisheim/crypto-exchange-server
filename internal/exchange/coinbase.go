// internal/exchange/coinbase.go
package exchange

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type CoinbaseClient struct {
	baseURL string
}

func NewCoinbaseClient() *CoinbaseClient {
	return &CoinbaseClient{
		baseURL: "https://api.exchange.coinbase.com",
	}
}

func (c *CoinbaseClient) GetOrderBook(symbol string) (*OrderBook, error) {
	url := fmt.Sprintf("%s/products/%s/book?level=2", c.baseURL, symbol)
	res, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", res.StatusCode)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	// Define the expected structure of the coinbase response,
	// coinbaseResponse.Bids/Asks are arrays of [price, size, num_orders]
	var coinbaseResponse struct {
		Bids [][3]interface{} `json:"bids"`
		Asks [][3]interface{} `json:"asks"`
	}

	err = json.Unmarshal(body, &coinbaseResponse)
	if err != nil {
		return nil, fmt.Errorf("error parsing json: %w", err)
	}

	// Convert Coinbase response to OrderBook format, third field (num_orders) is not needed
	orderBook := &OrderBook{
		Bids:     make([][2]string, len(coinbaseResponse.Bids)),
		Asks:     make([][2]string, len(coinbaseResponse.Asks)),
		Exchange: "coinbase",
	}

	for i, bid := range coinbaseResponse.Bids {
		orderBook.Bids[i] = [2]string{
			bid[0].(string), // Price
			bid[1].(string), // Size
		}
	}

	for i, ask := range coinbaseResponse.Asks {
		orderBook.Asks[i] = [2]string{
			ask[0].(string), // Price
			ask[1].(string), // Size
		}
	}

	return orderBook, nil
}
