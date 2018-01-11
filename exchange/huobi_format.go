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

type HuobiInfo struct {
	Status string `json:"status"`
	Data   struct {
		ID    uint64 `json:"id"`
		Type  string `json:"type"`
		State string `json:"state"`
		List  []struct {
			Currency string `json:"currency"`
			Type     string `json:"type"`
			Balance  string `json:"balance"`
		} `json:"list"`
	} `json:"data"`
}

type HuobiTrade struct {
	Status  string `json:"status"`
	OrderID string `json:"data"`
}

type HuobiCancel struct {
	Status  string `json:"status"`
	OrderID string `json:"data"`
}

type HuobiDeposit struct {
	Status  string `json:"status"`
	OrderID string `json:"data"`
}

type HuobiWithdraw struct {
	Status     string `json:"status"`
	WithdrawID uint64 `json:"data"`
}

type HuobiOrder struct {
	Status string `json:"status"`
	Data   struct {
		OrderID     uint64 `json:"id"`
		Symbol      string `json:"symbol"`
		AccountID   uint64 `json:"account-id"`
		OrigQty     string `json:"amount"`
		Price       string `json:"price"`
		Type        string `json:"type"`
		ExecutedQty string `json:"field-amount"`
	} `json:"data"`
}

type HuobiDepositAddress struct {
	Msg        string `json:"msg"`
	Address    string `json:"address"`
	Success    bool   `json:"success"`
	AddressTag string `json:"addressTag"`
	Asset      string `json:"asset"`
}

type HuobiAccounts struct {
	Status string `json:"status"`
	Data   []struct {
		ID     uint64 `json:"id"`
		Type   string `json:"type"`
		State  string `json:"state"`
		UserID uint64 `json:"user-id"`
	} `json:"data"`
}
