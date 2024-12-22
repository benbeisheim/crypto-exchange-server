// internal/exchange/coinbase.go
package exchange

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type CoinbaseClient struct{} // Empty struct, no fields needed

func NewCoinbaseClient() *CoinbaseClient {
	return &CoinbaseClient{}
}

func (c *CoinbaseClient) GetOrderBook(symbol string) (*OrderBook, error) {
	fmt.Println("symbol in CoinbaseClient.GetOrderBook", symbol)
	url := fmt.Sprintf("https://api.exchange.coinbase.com/products/%s/book?level=2", symbol)
	res, err := http.Get(url)
	if err != nil {
		fmt.Println("Error sending request", err)
		return nil, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return nil, err
	}

	var coinbaseResponse struct {
		Bids [][3]interface{} `json:"bids"`
		Asks [][3]interface{} `json:"asks"`
	}

	err = json.Unmarshal(body, &coinbaseResponse)
	if err != nil {
		fmt.Println("Error parsing json", err)
		return nil, err
	}

	fmt.Println("coinbaseResponse.Asks", len(coinbaseResponse.Asks))

	// Convert Coinbase response to new OrderBook format
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
	fmt.Println("orderBook.Asks length coinbase", len(orderBook.Asks))

	return orderBook, nil
}
