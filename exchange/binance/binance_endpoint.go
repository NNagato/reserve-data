package binance

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"math/big"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/KyberNetwork/reserve-data/common"
	"github.com/KyberNetwork/reserve-data/exchange"
	ethereum "github.com/ethereum/go-ethereum/common"
)

type BinanceEndpoint struct {
	signer Signer
	interf Interface
}

func (self *BinanceEndpoint) fillRequest(req *http.Request, signNeeded bool, timepoint uint64) {
	if req.Method == "POST" || req.Method == "PUT" || req.Method == "DELETE" {
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Add("User-Agent", "binance/go")
	}
	req.Header.Add("Accept", "application/json")
	if signNeeded {
		q := req.URL.Query()
		sig := url.Values{}
		req.Header.Set("X-MBX-APIKEY", self.signer.GetBinanceKey())
		q.Set("timestamp", fmt.Sprintf("%d", timepoint-5000))
		q.Set("recvWindow", "7000")
		sig.Set("signature", self.signer.BinanceSign(q.Encode()))
		// Using separated values map for signature to ensure it is at the end
		// of the query. This is required for /wapi apis from binance without
		// any damn documentation about it!!!
		req.URL.RawQuery = q.Encode() + "&" + sig.Encode()
	}
}

func (self *BinanceEndpoint) GetResponse(
	method string, url string,
	params map[string]string, signNeeded bool, timepoint uint64) ([]byte, error) {

	client := &http.Client{
		Timeout: time.Duration(30 * time.Second),
	}
	req, _ := http.NewRequest(method, url, nil)
	req.Header.Add("Accept", "application/json")

	q := req.URL.Query()
	for k, v := range params {
		q.Add(k, v)
	}
	req.URL.RawQuery = q.Encode()
	self.fillRequest(req, signNeeded, timepoint)
	var err error
	var resp_body []byte
	log.Printf("request to binance: %s\n", req.URL)
	resp, err := client.Do(req)
	if err != nil {
		return resp_body, err
	} else {
		defer resp.Body.Close()
		resp_body, err = ioutil.ReadAll(resp.Body)
		log.Printf("request to %s, got response from binance: %s\n", req.URL, common.TruncStr(resp_body))
		return resp_body, err
	}
}

func (self *BinanceEndpoint) GetDepthOnePair(
	pair common.TokenPair, timepoint uint64) (exchange.Binaresp, error) {

	resp_body, err := self.GetResponse(
		"GET", self.interf.PublicEndpoint()+"/api/v1/depth",
		map[string]string{
			"symbol": fmt.Sprintf("%s%s", pair.Base.ID, pair.Quote.ID),
			"limit":  "50",
		},
		false,
		timepoint,
	)

	resp_data := exchange.Binaresp{}
	if err != nil {
		return resp_data, err
	} else {
		json.Unmarshal(resp_body, &resp_data)
		return resp_data, nil
	}
}

// Relevant params:
// symbol ("%s%s", base, quote)
// side (BUY/SELL)
// type (LIMIT/MARKET)
// timeInForce (GTC/IOC)
// quantity
// price
//
// In this version, we only support LIMIT order which means only buy/sell with acceptable price,
// and GTC time in force which means that the order will be active until it's implicitly canceled
func (self *BinanceEndpoint) Trade(tradeType string, base, quote common.Token, rate, amount float64, timepoint uint64) (exchange.Binatrade, error) {
	result := exchange.Binatrade{}
	symbol := base.ID + quote.ID
	orderType := "LIMIT"
	params := map[string]string{
		"symbol":      symbol,
		"side":        strings.ToUpper(tradeType),
		"type":        orderType,
		"timeInForce": "GTC",
		"quantity":    strconv.FormatFloat(amount, 'f', -1, 64),
	}
	if orderType == "LIMIT" {
		params["price"] = strconv.FormatFloat(rate, 'f', -1, 64)
	}
	resp_body, err := self.GetResponse(
		"POST",
		self.interf.AuthenticatedEndpoint()+"/api/v3/order",
		params,
		true,
		timepoint,
	)
	if err != nil {
		return result, err
	} else {
		json.Unmarshal(resp_body, &result)
		return result, nil
	}
}

func (self *BinanceEndpoint) WithdrawHistory(startTime, endTime uint64) (exchange.Binawithdrawals, error) {
	result := exchange.Binawithdrawals{}
	timepoint := common.GetTimepoint()
	resp_body, err := self.GetResponse(
		"GET",
		self.interf.AuthenticatedEndpoint()+"/wapi/v3/withdrawHistory.html",
		map[string]string{
			"startTime": fmt.Sprintf("%d", startTime),
			"endTime":   fmt.Sprintf("%d", endTime),
		},
		true,
		timepoint,
	)
	if err == nil {
		json.Unmarshal(resp_body, &result)
		if !result.Success {
			err = errors.New("Getting withdraw history from Binance failed: " + result.Msg)
		}
	}
	return result, err
}

func (self *BinanceEndpoint) DepositHistory(startTime, endTime uint64) (exchange.Binadeposits, error) {
	result := exchange.Binadeposits{}
	timepoint := common.GetTimepoint()
	resp_body, err := self.GetResponse(
		"GET",
		self.interf.AuthenticatedEndpoint()+"/wapi/v3/depositHistory.html",
		map[string]string{
			"startTime": fmt.Sprintf("%d", startTime),
			"endTime":   fmt.Sprintf("%d", endTime),
		},
		true,
		timepoint,
	)
	if err == nil {
		err = json.Unmarshal(resp_body, &result)
		if !result.Success {
			err = errors.New("Getting deposit history from Binance failed: " + result.Msg)
		}
	}
	return result, err
}

func (self *BinanceEndpoint) CancelOrder(symbol string, id uint64) (exchange.Binacancel, error) {
	result := exchange.Binacancel{}
	resp_body, err := self.GetResponse(
		"DELETE",
		self.interf.AuthenticatedEndpoint()+"/api/v3/order",
		map[string]string{
			"symbol":  symbol,
			"orderId": fmt.Sprintf("%d", id),
		},
		true,
		common.GetTimepoint(),
	)
	if err == nil {
		json.Unmarshal(resp_body, &result)
		if result.Code != 0 {
			err = errors.New("Canceling order from Binance failed: " + result.Msg)
		}
	}
	return result, err
}

func (self *BinanceEndpoint) OrderStatus(symbol string, id uint64, timepoint uint64) (exchange.Binaorder, error) {
	result := exchange.Binaorder{}
	resp_body, err := self.GetResponse(
		"GET",
		self.interf.AuthenticatedEndpoint()+"/api/v3/order",
		map[string]string{
			"symbol":  symbol,
			"orderId": fmt.Sprintf("%d", id),
		},
		true,
		timepoint,
	)
	if err == nil {
		json.Unmarshal(resp_body, &result)
		if result.Code != 0 {
			err = errors.New(result.Message)
		}
	}
	return result, err
}

func (self *BinanceEndpoint) Withdraw(token common.Token, amount *big.Int, address ethereum.Address, timepoint uint64) (string, error) {
	result := exchange.Binawithdraw{}
	resp_body, err := self.GetResponse(
		"POST",
		self.interf.AuthenticatedEndpoint()+"/wapi/v3/withdraw.html",
		map[string]string{
			"asset":   token.ID,
			"address": address.Hex(),
			"name":    "reserve",
			"amount":  strconv.FormatFloat(common.BigToFloat(amount, token.Decimal), 'f', -1, 64),
		},
		true,
		timepoint,
	)
	if err == nil {
		json.Unmarshal(resp_body, &result)
		if result.Success == false {
			return "", errors.New(result.Message)
		}
		return result.ID, nil
	} else {
		log.Printf("Error: %v", err)
		return "", errors.New("withdraw rejected by Binance")
	}
}

func (self *BinanceEndpoint) GetInfo(timepoint uint64) (exchange.Binainfo, error) {
	result := exchange.Binainfo{}
	resp_body, err := self.GetResponse(
		"GET",
		self.interf.AuthenticatedEndpoint()+"/api/v3/account",
		map[string]string{},
		true,
		timepoint,
	)
	if err == nil {
		json.Unmarshal(resp_body, &result)
	}
	return result, err
}

func (self *BinanceEndpoint) OpenOrdersForOnePair(
	pair common.TokenPair, timepoint uint64) (exchange.Binaorders, error) {

	result := exchange.Binaorders{}
	resp_body, err := self.GetResponse(
		"GET",
		self.interf.AuthenticatedEndpoint()+"/api/v3/openOrders",
		map[string]string{
			"symbol": pair.Base.ID + pair.Quote.ID,
		},
		true,
		timepoint,
	)
	if err != nil {
		return result, err
	} else {
		json.Unmarshal(resp_body, &result)
		return result, nil
	}
}

func (self *BinanceEndpoint) GetDepositAddress(asset string) (exchange.Binadepositaddress, error) {
	result := exchange.Binadepositaddress{}
	timepoint := common.GetTimepoint()
	resp_body, err := self.GetResponse(
		"GET",
		self.interf.AuthenticatedEndpoint()+"/wapi/v3/depositAddress.html",
		map[string]string{
			"asset": asset,
		},
		true,
		timepoint,
	)
	if err == nil {
		err = json.Unmarshal(resp_body, &result)
		if !result.Success {
			err = errors.New(result.Msg)
		}
	}
	return result, err
}

func (self *BinanceEndpoint) GetExchangeInfo() (exchange.BinanceExchangeInfo, error) {
	result := exchange.BinanceExchangeInfo{}
	timepoint := common.GetTimepoint()
	resp_body, err := self.GetResponse(
		"GET",
		self.interf.PublicEndpoint()+"/api/v1/exchangeInfo",
		map[string]string{},
		false,
		timepoint,
	)
	if err == nil {
		err = json.Unmarshal(resp_body, &result)
	}
	return result, err
}

func NewBinanceEndpoint(signer Signer, interf Interface) *BinanceEndpoint {
	return &BinanceEndpoint{signer, interf}
}

func NewRealBinanceEndpoint(signer Signer) *BinanceEndpoint {
	return &BinanceEndpoint{signer, NewRealInterface()}
}

func NewSimulatedBinanceEndpoint(signer Signer) *BinanceEndpoint {
	return &BinanceEndpoint{signer, NewSimulatedInterface()}
}

func NewRopstenBinanceEndpoint(signer Signer) *BinanceEndpoint {
	return &BinanceEndpoint{signer, NewRopstenInterface()}
}

func NewDevBinanceEndpoint(signer Signer) *BinanceEndpoint {
	return &BinanceEndpoint{signer, NewDevInterface()}
}
