// internal/exchange/kraken.go
package exchange

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type KrakenClient struct{}

func NewKrakenClient() *KrakenClient {
	return &KrakenClient{}
}

func (c *KrakenClient) GetOrderBook(symbol string) (*OrderBook, error) {
	url := fmt.Sprintf("https://api.kraken.com/0/public/Depth?pair=%s", symbol)

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Add("User-Agent", "Mozilla/5.0")
	req.Header.Add("Accept", "application/json")

	res, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending request:", err)
		return nil, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Println("Error reading response body")
		return nil, err
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

	// Unmarshal into the full response struct
	err = json.Unmarshal(body, &krakenResponse)
	if err != nil {
		fmt.Printf("Error parsing JSON: %v\n", err)
		fmt.Printf("Raw JSON: %s\n", string(body))
		return nil, err
	}

	// Check for any errors in the response
	if len(krakenResponse.Error) > 0 {
		fmt.Println("Error in kraken API response")
	}

	// Get the first (and typically only) order book in the result
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
