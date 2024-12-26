package exchange

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type KrakenClient struct {
	baseURL string
}

func NewKrakenClient() *KrakenClient {
	return &KrakenClient{
		baseURL: "https://api.kraken.com",
	}
}

func (c *KrakenClient) GetOrderBook(symbol string) (*OrderBook, error) {
	krakenSymbol := strings.ReplaceAll(symbol, "-", "")
	url := fmt.Sprintf("%s/0/public/Depth?pair=%s", c.baseURL, krakenSymbol)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Add("User-Agent", "Mozilla/5.0")
	req.Header.Add("Accept", "application/json")

	res, err := http.DefaultClient.Do(req)
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

	// Create a struct that matches the full Kraken API response

	type krakenOrderBook struct {
		Asks [][3]interface{} `json:"asks"`
		Bids [][3]interface{} `json:"bids"`
	}

	var krakenResponse struct {
		Error  []string                   `json:"error"`
		Result map[string]krakenOrderBook `json:"result"`
	}

	// Unmarshal into the krakenResponse struct
	err = json.Unmarshal(body, &krakenResponse)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling response: %w", err)
	}

	// Check for any errors in the response
	if len(krakenResponse.Error) > 0 {
		return nil, fmt.Errorf("error in kraken API response: %v", krakenResponse.Error)
	}

	// Get the order book data from the result, krakenResponse.Result is a map of pairs requested to
	// order book data, since we only request one pair, we take the first (only) one
	var orderBookData krakenOrderBook
	for _, book := range krakenResponse.Result {
		orderBookData = book
		break
	}

	// Convert Kraken response to OrderBook format
	orderBook := &OrderBook{
		Bids:     make([][2]string, len(orderBookData.Bids)),
		Asks:     make([][2]string, len(orderBookData.Asks)),
		Exchange: "kraken",
	}

	for i, bid := range orderBookData.Bids {
		orderBook.Bids[i] = [2]string{
			bid[0].(string), // Price
			bid[1].(string), // Size
		}
	}

	for i, ask := range orderBookData.Asks {
		orderBook.Asks[i] = [2]string{
			ask[0].(string), // Price
			ask[1].(string), // Size
		}
	}

	return orderBook, nil
}
