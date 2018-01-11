package exchange

type HuobiDepth struct {
	Status    string `json:"status"`
	Timestamp uint64 `json:"ts"`
	Tick      struct {
		Bids [][]float64 `json:"bids"`
		Asks [][]float64 `json:"asks"`
	} `json:"tick"`
}

type HuobiExchangeInfo struct {
	Status string `json:"status"`
	Data   []struct {
		Base            string `json:"base-currency"`
		Quote           string `json:"quote-currency"`
		PricePrecision  int    `json:"price-precision"`
		AmountPrecision int    `json:"amount-precision"`
	} `json:"data"`
}

type HuobiTrade struct {
	Status  string `json:"status"`
	OrderID string `json:"data"`
}

type HuobiWithdraw struct {
	Status     string `json:"status"`
	WithdrawID uint64 `json:"data"`
}
