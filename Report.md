# Report

## 1. If you had more time, what further improvements or new features would you add?

### Order Book Retrieval

- Currently requesting full order book data (both bids and asks). It seems unfortunate that every time we request order book data, we must necessarily request unneeded data. A brief look into the Kraken API documentation suggests that one sided market data cannot be requested via REST API.

### Testing 
- Should be testing for edge cases where ticker naming is not consistent across exchanges. EX: XBT vs BTC. Saw that kraken API accepts BTCUSD as pair parameter although native ticker is XBTUSD. Did not yet look into other potential ticker naming inconsistencies.  

### Additional Features

- Could add additional exchanges to the project. 
- Could add pair naming normalization so that the client can input various formats rather than just Coinbase hyphenated format. Could also address ticker naming normalization rather than relying on kraken to recognize non native tickers. 
- Add concurrency to the order book retrieval process. 

## 2. Which parts are you most proud of? And why?

- I am most proud of the testing. I am proud that I was able to grasp the basics of mocking, as this was the first time I had ever used it. My most proud moment was the first time I ran the full test suite and saw that finally, everything was OK. I am also proud of the order book aggregation logic, because while it still retains plenty of room for improvement, it is far better than the initial naive implementation I came up with. The current implementation is far more efficient as it has abstracted out common logic for buy/sell operations in the service layer.

## 3.  Which parts did you spend the most time with? What did you find most difficult?

- I spent the most time with the testing. I found this difficult because testing was the concept that I was the least familiar with going into the project. 

## 4. How did you find the test overall? Did you have any issues or have difficulties completing? If you have any suggestions on how we can improve the test, we'd love to hear them.

- I found the test overall to be a good challange for me as well as an invaluable learning experience. I was pleased to find that achieving a working solution in a language I had never used before, within a few days, was possible. Learning the basics of a relatively lower level language (to python and javascript with which I am familiar) provided me with an appreciation for the value of concepts such as type safety. I was surprised to find that the more opinionated nature of go turned out to be aesthetically pleasing to me, as I had only been able to understand such opinions as tedium, before being exposed to their value in this project. 