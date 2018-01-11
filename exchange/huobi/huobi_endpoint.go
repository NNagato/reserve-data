package huobi

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

type HuobiEndpoint struct {
	signer Signer
	interf Interface
}

func (self *HuobiEndpoint) fillRequest(req *http.Request, signNeeded bool, timepoint uint64) {
	if req.Method == "POST" || req.Method == "PUT" || req.Method == "DELETE" {
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	}
	req.Header.Add("Accept", "application/json")
	if signNeeded {
		q := req.URL.Query()
		sig := url.Values{}
		sig.Set("Signature", self.signer.HuobiSign(q.Encode()))
		// Using separated values map for signature to ensure it is at the end
		// of the query. This is required for /wapi apis from huobi without
		// any damn documentation about it!!!
		req.URL.RawQuery = q.Encode() + "&" + sig.Encode()
	}
}

func (self *HuobiEndpoint) GetResponse(
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
	log.Printf("request to huobi: %s\n", req.URL)
	resp, err := client.Do(req)
	if err != nil {
		return resp_body, err
	} else {
		defer resp.Body.Close()
		resp_body, err = ioutil.ReadAll(resp.Body)
		log.Printf("request to %s, got response from huobi: %s\n", req.URL, common.TruncStr(resp_body))
		return resp_body, err
	}
}

func (self *HuobiEndpoint) GetDepthOnePair(
	pair common.TokenPair, timepoint uint64) (exchange.HuobiDepth, error) {

	resp_body, err := self.GetResponse(
		"GET", self.interf.PublicEndpoint()+"/market/depth",
		map[string]string{
			"symbol": fmt.Sprintf("%s%s", strings.ToLower(pair.Base.ID), strings.ToLower(pair.Quote.ID)),
			"type":   "step0",
		},
		false,
		timepoint,
	)

	resp_data := exchange.HuobiDepth{}
	if err != nil {
		return resp_data, err
	} else {
		json.Unmarshal(resp_body, &resp_data)
		return resp_data, nil
	}
}

func (self *HuobiEndpoint) Trade(tradeType string, base, quote common.Token, rate, amount float64, timepoint uint64) (exchange.HuobiTrade, error) {
	result := exchange.HuobiTrade{}
	symbol := base.ID + quote.ID
	orderType := tradeType + "-limit"
	params := map[string]string{
		"account_id": "", //TODO: get account id
		"symbol":     symbol,
		"side":       strings.ToUpper(tradeType),
		"type":       orderType,
		"amount":     strconv.FormatFloat(amount, 'f', -1, 64),
	}
	if orderType == "buy-limit" {
		params["price"] = strconv.FormatFloat(rate, 'f', -1, 64)
	}
	resp_body, err := self.GetResponse(
		"POST",
		self.interf.AuthenticatedEndpoint()+"/v1/order/orders/place",
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

func (self *HuobiEndpoint) WithdrawHistory(startTime, endTime uint64) (exchange.HuobiWithdraw, error) {
	result := exchange.HuobiWithdraw{}
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
		if result.Status != "ok" {
			err = errors.New("Getting withdraw history from Huobi failed")
		}
	}
	return result, err
}

func (self *HuobiEndpoint) DepositHistory(startTime, endTime uint64) (exchange.HuobiDeposit, error) {
	result := exchange.HuobiDeposit{}
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
		if result.Status != "ok" {
			err = errors.New("Getting deposit history from Huobi failed")
		}
	}
	return result, err
}

func (self *HuobiEndpoint) CancelOrder(symbol string, id uint64) (exchange.HuobiCancel, error) {
	result := exchange.HuobiCancel{}
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
		if result.Status != "ok" {
			err = errors.New("Canceling order from Huobi failed")
		}
	}
	return result, err
}

func (self *HuobiEndpoint) OrderStatus(symbol string, id uint64, timepoint uint64) (exchange.HuobiOrder, error) {
	result := exchange.HuobiOrder{}
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
		if result.Status != "ok" {
			err = errors.New("Get order form Huobi failed")
		}
	}
	return result, err
}

func (self *HuobiEndpoint) Withdraw(token common.Token, amount *big.Int, address ethereum.Address, timepoint uint64) (string, error) {
	result := exchange.HuobiWithdraw{}
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
		if result.Status != "ok" {
			return "", errors.New("Cannot withdraw from huobi")
		}
		return string(result.WithdrawID), nil
	} else {
		log.Printf("Error: %v", err)
		return "", errors.New("withdraw rejected by Huobi")
	}
}

func (self *HuobiEndpoint) GetInfo(timepoint uint64) (exchange.HuobiInfo, error) {
	result := exchange.HuobiInfo{}
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

func (self *HuobiEndpoint) OpenOrdersForOnePair(
	pair common.TokenPair, timepoint uint64) (exchange.HuobiOrder, error) {

	result := exchange.HuobiOrder{}
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

func (self *HuobiEndpoint) GetDepositAddress(asset string) (exchange.HuobiDepositAddress, error) {
	result := exchange.HuobiDepositAddress{}
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

func (self *HuobiEndpoint) GetExchangeInfo() (exchange.HuobiExchangeInfo, error) {
	result := exchange.HuobiExchangeInfo{}
	timepoint := common.GetTimepoint()
	resp_body, err := self.GetResponse(
		"GET",
		self.interf.PublicEndpoint()+"/v1/common/symbols",
		map[string]string{},
		false,
		timepoint,
	)
	if err == nil {
		err = json.Unmarshal(resp_body, &result)
	}
	return result, err
}

func NewHuobiEndpoint(signer Signer, interf Interface) *HuobiEndpoint {
	return &HuobiEndpoint{signer, interf}
}

func NewRealHuobiEndpoint(signer Signer) *HuobiEndpoint {
	return &HuobiEndpoint{signer, NewRealInterface()}
}

func NewSimulatedHuobiEndpoint(signer Signer) *HuobiEndpoint {
	return &HuobiEndpoint{signer, NewSimulatedInterface()}
}

func NewRopstenHuobiEndpoint(signer Signer) *HuobiEndpoint {
	return &HuobiEndpoint{signer, NewRopstenInterface()}
}

func NewDevHuobiEndpoint(signer Signer) *HuobiEndpoint {
	return &HuobiEndpoint{signer, NewDevInterface()}
}
