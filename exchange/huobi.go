package exchange

import (
	"errors"
	"fmt"
	"log"
	"math/big"
	"strconv"
	"strings"
	"sync"

	"github.com/KyberNetwork/reserve-data/common"
	ethereum "github.com/ethereum/go-ethereum/common"
)

const HUOBI_EPSILON float64 = 0.0000000001 // 10e-10

type Huobi struct {
	interf       HuobiInterface
	pairs        []common.TokenPair
	addresses    map[string]ethereum.Address
	exchangeInfo *common.ExchangeInfo
	fees         common.ExchangeFees
}

func (self *Huobi) MarshalText() (text []byte, err error) {
	return []byte(self.ID()), nil
}

func (self *Huobi) Address(token common.Token) (ethereum.Address, bool) {
	addr, supported := self.addresses[token.ID]
	return addr, supported
}

func (self *Huobi) UpdateAllDepositAddresses(address string) {
	for k, _ := range self.addresses {
		self.addresses[k] = ethereum.HexToAddress(address)
	}
}

func (self *Huobi) UpdateDepositAddress(token common.Token, address string) {
	self.addresses[token.ID] = ethereum.HexToAddress(address)
}

func (self *Huobi) UpdatePrecisionLimit(pair common.TokenPair, symbols HuobiExchangeInfo) {
	pairName := strings.ToLower(pair.Base.ID) + strings.ToLower(pair.Quote.ID)
	for _, symbol := range symbols.Data {
		if symbol.Base+symbol.Quote == pairName {
			exchangePrecisionLimit := common.ExchangePrecisionLimit{}
			exchangePrecisionLimit.Precision.Amount = symbol.AmountPrecision
			exchangePrecisionLimit.Precision.Price = symbol.PricePrecision
			self.exchangeInfo.Update(pair.PairID(), exchangePrecisionLimit)
			break
		}
	}
}

func (self *Huobi) UpdatePairsPrecision() {
	exchangeInfo, err := self.interf.GetExchangeInfo()
	if err != nil {
		log.Printf("Get exchange info failed: %s\n", err)
	} else {
		for _, pair := range self.pairs {
			self.UpdatePrecisionLimit(pair, exchangeInfo)
		}
	}
}

func (self *Huobi) GetInfo() (common.ExchangeInfo, error) {
	return *self.exchangeInfo, nil
}

func (self *Huobi) GetExchangeInfo(pair common.TokenPairID) (common.ExchangePrecisionLimit, error) {
	data, err := self.exchangeInfo.Get(pair)
	return data, err
}

func (self *Huobi) GetFee() common.ExchangeFees {
	return self.fees
}

func (self *Huobi) ID() common.ExchangeID {
	return common.ExchangeID("huobi")
}

func (self *Huobi) TokenPairs() []common.TokenPair {
	return self.pairs
}

func (self *Huobi) Name() string {
	return "huobi"
}

func (self *Huobi) QueryOrder(symbol string, id uint64, timepoint uint64) (done float64, remaining float64, finished bool, err error) {
	result, err := self.interf.OrderStatus(symbol, id, timepoint)
	if err != nil {
		return 0, 0, false, err
	} else {
		done, _ := strconv.ParseFloat(result.Data.ExecutedQty, 64)
		total, _ := strconv.ParseFloat(result.Data.OrigQty, 64)
		return done, total - done, total-done < HUOBI_EPSILON, nil
	}
}

func (self *Huobi) Trade(tradeType string, base common.Token, quote common.Token, rate float64, amount float64, timepoint uint64) (id string, done float64, remaining float64, finished bool, err error) {
	result, err := self.interf.Trade(tradeType, base, quote, rate, amount, timepoint)
	symbol := base.ID + quote.ID

	if err != nil {
		return "", 0, 0, false, err
	} else {
		orderID, _ := strconv.ParseUint(result.OrderID, 10, 64)
		done, remaining, finished, err := self.QueryOrder(
			base.ID+quote.ID,
			orderID,
			timepoint+20,
		)
		id := fmt.Sprintf("%s_%s", result.OrderID, symbol)
		return id, done, remaining, finished, err
	}
}

func (self *Huobi) Withdraw(token common.Token, amount *big.Int, address ethereum.Address, timepoint uint64) (string, error) {
	tx, err := self.interf.Withdraw(token, amount, address, timepoint)
	return tx, err
}

func (self *Huobi) CancelOrder(id common.ActivityID) error {
	idParts := strings.Split(id.EID, "_")
	idNo, err := strconv.ParseUint(idParts[0], 10, 64)
	if err != nil {
		return err
	}
	symbol := idParts[1]
	result, err := self.interf.CancelOrder(symbol, idNo)
	if err != nil {
		return err
	}
	if result.Status != "ok" {
		return errors.New("Couldn't cancel order id " + id.EID)
	}
	return nil
}

func (self *Huobi) FetchOnePairData(
	wg *sync.WaitGroup,
	pair common.TokenPair,
	data *sync.Map,
	timepoint uint64) {

	defer wg.Done()
	result := common.ExchangePrice{}

	timestamp := common.Timestamp(fmt.Sprintf("%d", timepoint))
	result.Timestamp = timestamp
	result.Valid = true
	resp_data, err := self.interf.GetDepthOnePair(pair, timepoint)
	returnTime := common.GetTimestamp()
	result.ReturnTime = returnTime
	if err != nil {
		result.Valid = false
		result.Error = err.Error()
	} else {
		if resp_data.Status != "ok" {
			result.Valid = false
		} else {
			for _, buy := range resp_data.Tick.Bids {
				quantity := buy[1]
				rate := buy[0]
				result.Bids = append(
					result.Bids,
					common.PriceEntry{
						quantity,
						rate,
					},
				)
			}
			for _, sell := range resp_data.Tick.Asks {
				quantity := sell[1]
				rate := sell[0]
				result.Asks = append(
					result.Asks,
					common.PriceEntry{
						quantity,
						rate,
					},
				)
			}
		}
	}
	data.Store(pair.PairID(), result)
}

func (self *Huobi) FetchPriceData(timepoint uint64) (map[common.TokenPairID]common.ExchangePrice, error) {
	wait := sync.WaitGroup{}
	data := sync.Map{}
	pairs := self.pairs
	for _, pair := range pairs {
		wait.Add(1)
		go self.FetchOnePairData(&wait, pair, &data, timepoint)
	}
	wait.Wait()
	result := map[common.TokenPairID]common.ExchangePrice{}
	data.Range(func(key, value interface{}) bool {
		result[key.(common.TokenPairID)] = value.(common.ExchangePrice)
		return true
	})
	return result, nil
}

func (self *Huobi) OpenOrdersForOnePair(
	wg *sync.WaitGroup,
	pair common.TokenPair,
	data *sync.Map,
	timepoint uint64) {

	// defer wg.Done()

	// result, err := self.interf.OpenOrdersForOnePair(pair, timepoint)

	//TODO: complete open orders for one pair
}

func (self *Huobi) FetchOrderData(timepoint uint64) (common.OrderEntry, error) {
	result := common.OrderEntry{}
	result.Timestamp = common.Timestamp(fmt.Sprintf("%d", timepoint))
	result.Valid = true
	result.Data = []common.Order{}

	wait := sync.WaitGroup{}
	data := sync.Map{}
	pairs := self.pairs
	for _, pair := range pairs {
		wait.Add(1)
		go self.OpenOrdersForOnePair(&wait, pair, &data, timepoint)
	}
	wait.Wait()

	result.ReturnTime = common.GetTimestamp()

	data.Range(func(key, value interface{}) bool {
		orders := value.([]common.Order)
		result.Data = append(result.Data, orders...)
		return true
	})
	return result, nil
}

func (self *Huobi) FetchEBalanceData(timepoint uint64) (common.EBalanceEntry, error) {
	result := common.EBalanceEntry{}
	result.Timestamp = common.Timestamp(fmt.Sprintf("%d", timepoint))
	result.Valid = true
	resp_data, err := self.interf.GetInfo(timepoint)
	result.ReturnTime = common.GetTimestamp()
	if err != nil {
		result.Valid = false
		result.Error = err.Error()
	} else {
		result.AvailableBalance = map[string]float64{}
		result.LockedBalance = map[string]float64{}
		result.DepositBalance = map[string]float64{}
		if resp_data.Status != "ok" {
			result.Valid = false
			result.Error = fmt.Sprintf("Cannot fetch ebalance")
		} else {
			// TODO: update ebalance data
			return result, nil
		}
	}
	return result, nil
}

func (self *Huobi) DepositStatus(id common.ActivityID, timepoint uint64) (string, error) {
	// TODO: complete this deposit status
	return "", errors.New("Deposit doesn't exist. This shouldn't happen unless tx returned from huobi and activity ID are not consistently designed")
}

func (self *Huobi) WithdrawStatus(id common.ActivityID, timepoint uint64) (string, string, error) {
	// TODO: complete this withdraw status
	return "", "", errors.New("Withdrawal doesn't exist. This shouldn't happen unless tx returned from withdrawal from huobi and activity ID are not consistently designed")
}

func (self *Huobi) OrderStatus(id common.ActivityID, timepoint uint64) (string, error) {
	tradeID := id.EID
	parts := strings.Split(tradeID, "_")
	orderID, err := strconv.ParseUint(parts[0], 10, 64)
	if err != nil {
		// if this crashes, it means core put malformed activity ID
		panic(err)
	}
	symbol := parts[1]
	order, err := self.interf.OrderStatus(symbol, orderID, timepoint)
	if err != nil {
		return "", err
	}
	if order.Status == "pre-submitted" || order.Status == "submitting" || order.Status == "submitted" || order.Status == "partial-filled" || order.Status == "partial-canceled" {
		return "", nil
	} else {
		return "done", nil
	}
}

func NewHuobi(interf HuobiInterface) *Huobi {
	return &Huobi{
		interf,
		[]common.TokenPair{
			common.MustCreateTokenPair("MCO", "ETH"),
			common.MustCreateTokenPair("OMG", "ETH"),
			common.MustCreateTokenPair("EOS", "ETH"),
			common.MustCreateTokenPair("KNC", "ETH"),
		},
		map[string]ethereum.Address{},
		common.NewExchangeInfo(),
		common.NewExchangeFee(
			common.TradingFee{
				"taker": 0.002,
				"maker": 0.002,
			},
			common.NewFundingFee(
				map[string]float32{
					"ETH": 0.01,
					"EOS": 0.5,
					"OMG": 0.1,
					"KNC": 1.0,
					"MCO": 0.2,
				},
				map[string]float32{
					"ETH": 0,
					"EOS": 0,
					"OMG": 0,
					"KNC": 0,
					"MCO": 0,
				},
			),
		),
	}
}
