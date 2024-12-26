# Crypto Exchange Service

This service is a simple API that allows you to get the best price for a given crypto pair across the two supported exchanges: Coinbase and Kraken.

## Endpoints

- Buy: http://localhost:4000/buy?amount=1&symbol=BTC-USDT -- returns buy order for 1 BTC in BTC-USDT market
- Sell: http://localhost:4000/sell?amount=1&symbol=BTC-USDT -- returns sell order for 1 BTC in BTC-USDT market
- Symbol format must be BASE-QUOTE (- seperates base and quote)

### Running the service

- Run the service with the following command from the root directory: `go run cmd/server/main.go`
- Make get requests to the endpoints above

### Testing

- Run the tests with the following command from the root directory: `go test ./...`
- Test suite covers service, controller, and exchange packages. 
- Run tests for a specific package with the following command: `go test ./internal/service`



