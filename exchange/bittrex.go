package exchange

import (
	"errors"
	"fmt"
	"log"
	"math/big"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/KyberNetwork/reserve-data/common"
	ethereum "github.com/ethereum/go-ethereum/common"
)

const BITTREX_EPSILON float64 = 0.00000001

type Bittrex struct {
	interf       BittrexInterface
	pairs        []common.TokenPair
	addresses    map[string]ethereum.Address
	storage      BittrexStorage
	exchangeInfo *common.ExchangeInfo
	fees         common.ExchangeFees
}

func (self *Bittrex) MarshalText() (text []byte, err error) {
	return []byte(self.ID()), nil
}

func (self *Bittrex) Address(token common.Token) (ethereum.Address, bool) {
	addr, supported := self.addresses[token.ID]
	return addr, supported
}

func (self *Bittrex) GetFee() common.ExchangeFees {
	return self.fees
}

func (self *Bittrex) UpdateAllDepositAddresses(address string) {
	for k, _ := range self.addresses {
		self.addresses[k] = ethereum.HexToAddress(address)
	}
}

func (self *Bittrex) UpdateDepositAddress(token common.Token, address string) {
	self.addresses[token.ID] = ethereum.HexToAddress(address)
}

func (self *Bittrex) UpdatePrecisionLimit(pair common.TokenPair, symbols []BittPairInfo) {
	pairName := strings.ToUpper(pair.Base.ID) + strings.ToUpper(pair.Quote.ID)
	for _, symbol := range symbols {
		symbolName := strings.ToUpper(symbol.Base) + strings.ToUpper(symbol.Quote)
		if symbolName == pairName {
			exchangePrecisionLimit := common.ExchangePrecisionLimit{}
			//update precision
			exchangePrecisionLimit.Precision.Amount = 8
			exchangePrecisionLimit.Precision.Price = 8
			// update limit
			exchangePrecisionLimit.AmountLimit.Min = symbol.MinAmount
			self.exchangeInfo.Update(pair.PairID(), exchangePrecisionLimit)
			break
		}
	}
}

func (self *Bittrex) GetExchangeInfo(pair common.TokenPairID) (common.ExchangePrecisionLimit, error) {
	pairInfo, err := self.exchangeInfo.Get(pair)
	return pairInfo, err
}

func (self *Bittrex) GetInfo() (common.ExchangeInfo, error) {
	return *self.exchangeInfo, nil
}

func (self *Bittrex) UpdatePairsPrecision() {
	exchangeInfo, err := self.interf.GetExchangeInfo()
	if err == nil {
		symbols := exchangeInfo.Pairs
		for _, pair := range self.pairs {
			self.UpdatePrecisionLimit(pair, symbols)
		}
	} else {
		log.Printf("Get exchange info failed: %s\n", err)
	}
}

func (self *Bittrex) ID() common.ExchangeID {
	return common.ExchangeID("bittrex")
}

func (self *Bittrex) TokenPairs() []common.TokenPair {
	return self.pairs
}

func (self *Bittrex) Name() string {
	return "bittrex"
}

func (self *Bittrex) QueryOrder(uuid string, timepoint uint64) (float64, float64, bool, error) {
	result, err := self.interf.OrderStatus(uuid, timepoint)
	if err != nil {
		return 0, 0, false, err
	} else {
		remaining := result.Result.QuantityRemaining
		done := result.Result.Quantity - remaining
		return done, remaining, remaining < BITTREX_EPSILON, nil
	}
}

func (self *Bittrex) Trade(tradeType string, base common.Token, quote common.Token, rate float64, amount float64, timepoint uint64) (string, float64, float64, bool, error) {
	result, err := self.interf.Trade(tradeType, base, quote, rate, amount, timepoint)

	if err != nil {
		return "", 0, 0, false, errors.New("Trade rejected by Bittrex")
	} else {
		if result.Success {
			uuid := result.Result["uuid"]
			done, remaining, finished, err := self.QueryOrder(
				uuid, timepoint+20)
			return uuid, done, remaining, finished, err
		} else {
			return "", 0, 0, false, errors.New(result.Error)
		}
	}
}

func (self *Bittrex) Withdraw(token common.Token, amount *big.Int, address ethereum.Address, timepoint uint64) (string, error) {
	resp, err := self.interf.Withdraw(token, amount, address, timepoint)
	if err != nil {
		return "", err
	} else {
		if resp.Success {
			return resp.Result["uuid"] + "|" + token.ID, nil
		} else {
			return "", errors.New(resp.Error)
		}
	}
}

func bitttimestampToUint64(input string) uint64 {
	var t time.Time
	var err error
	len := len(input)
	if len == 23 {
		t, err = time.Parse("2006-01-02T15:04:05.000", input)
	} else if len == 22 {
		t, err = time.Parse("2006-01-02T15:04:05.00", input)
	} else if len == 21 {
		t, err = time.Parse("2006-01-02T15:04:05.0", input)
	}
	if err != nil {
		panic(err)
	}
	return uint64(t.UnixNano() / int64(time.Millisecond))
}

func (self *Bittrex) DepositStatus(id common.ActivityID, timepoint uint64) (string, error) {
	timestamp := id.Timepoint
	idParts := strings.Split(id.EID, "|")
	if len(idParts) != 3 {
		// here, the exchange id part in id is malformed
		// 1. because analytic didn't pass original ID
		// 2. id is not constructed correctly in a form of uuid + "|" + token + "|" + amount
		return "", errors.New("Invalid deposit id")
	}
	currency := idParts[1]
	amount, err := strconv.ParseFloat(idParts[2], 64)
	if err != nil {
		panic(err)
	}
	histories, err := self.interf.DepositHistory(currency, timepoint)
	if err != nil {
		return "", err
	} else {
		for _, deposit := range histories.Result {
			if deposit.Currency == currency &&
				deposit.Amount-amount < BITTREX_EPSILON &&
				bitttimestampToUint64(deposit.LastUpdated) > timestamp/uint64(time.Millisecond) &&
				self.storage.IsNewBittrexDeposit(deposit.Id, id) {
				self.storage.RegisterBittrexDeposit(deposit.Id, id)
				return "done", nil
			}
		}
		return "", nil
	}
}

func (self *Bittrex) CancelOrder(id common.ActivityID) error {
	uuid := id.EID
	resp, err := self.interf.CancelOrder(uuid, common.GetTimepoint())
	if err != nil {
		return err
	} else {
		if resp.Success {
			return nil
		} else {
			return errors.New(resp.Error)
		}
	}
}

func (self *Bittrex) WithdrawStatus(id common.ActivityID, timepoint uint64) (string, string, error) {
	idParts := strings.Split(id.EID, "|")
	if len(idParts) != 2 {
		// here, the exchange id part in id is malformed
		// 1. because analytic didn't pass original ID
		// 2. id is not constructed correctly in a form of uuid + "|" + token
		return "", "", errors.New("Invalid withdraw id")
	}
	uuid := idParts[0]
	currency := idParts[1]
	histories, err := self.interf.WithdrawHistory(currency, timepoint)
	if err != nil {
		return "", "", err
	} else {
		for _, withdraw := range histories.Result {
			if withdraw.PaymentUuid == uuid {
				if withdraw.PendingPayment {
					return "", withdraw.TxId, nil
				} else {
					return "done", withdraw.TxId, nil
				}
			}
		}
		return "", "", errors.New("Withdraw with uuid " + uuid + " of currency " + currency + " is not found on bittrex")
	}
}

func (self *Bittrex) OrderStatus(id common.ActivityID, timepoint uint64) (string, error) {
	uuid := id.EID
	resp_data, err := self.interf.OrderStatus(uuid, timepoint)
	if err != nil {
		return "", err
	} else {
		if resp_data.Result.IsOpen {
			return "", nil
		} else {
			return "done", nil
		}
	}
}

func (self *Bittrex) FetchOnePairData(wq *sync.WaitGroup, pair common.TokenPair, data *sync.Map, timepoint uint64) {
	defer wq.Done()
	result := common.ExchangePrice{}
	result.Timestamp = common.Timestamp(fmt.Sprintf("%d", timepoint))
	result.Valid = true
	onePairData, err := self.interf.FetchOnePairData(pair, timepoint)
	returnTime := common.GetTimestamp()
	result.ReturnTime = returnTime
	if err != nil {
		result.Valid = false
		result.Error = err.Error()
	} else {
		if !onePairData.Success {
			result.Valid = false
			result.Error = onePairData.Msg
		} else {
			for _, buy := range onePairData.Result["buy"] {
				result.Bids = append(
					result.Bids,
					common.PriceEntry{
						buy["Quantity"],
						buy["Rate"],
					},
				)
			}
			for _, sell := range onePairData.Result["sell"] {
				result.Asks = append(
					result.Asks,
					common.PriceEntry{
						sell["Quantity"],
						sell["Rate"],
					},
				)
			}
		}
	}
	data.Store(pair.PairID(), result)
}

func (self *Bittrex) FetchPriceData(timepoint uint64) (map[common.TokenPairID]common.ExchangePrice, error) {
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

func (self *Bittrex) FetchEBalanceData(timepoint uint64) (common.EBalanceEntry, error) {
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
		if resp_data.Success {
			for _, b := range resp_data.Result {
				tokenID := b.Currency
				_, exist := common.SupportedTokens[tokenID]
				if exist {
					result.AvailableBalance[tokenID] = b.Available
					result.DepositBalance[tokenID] = b.Pending
					result.LockedBalance[tokenID] = 0
				}
			}
		} else {
			result.Valid = false
			result.Error = resp_data.Error
		}
	}
	return result, nil
}

func NewBittrex(interf BittrexInterface, storage BittrexStorage) *Bittrex {
	return &Bittrex{
		interf,
		[]common.TokenPair{
			common.MustCreateTokenPair("OMG", "ETH"),
			common.MustCreateTokenPair("DGD", "ETH"),
			common.MustCreateTokenPair("CVC", "ETH"),
			common.MustCreateTokenPair("FUN", "ETH"),
			common.MustCreateTokenPair("MCO", "ETH"),
			common.MustCreateTokenPair("GNT", "ETH"),
			common.MustCreateTokenPair("ADX", "ETH"),
			common.MustCreateTokenPair("PAY", "ETH"),
			common.MustCreateTokenPair("BAT", "ETH"),
		},
		map[string]ethereum.Address{},
		storage,
		common.NewExchangeInfo(),
		common.NewExchangeFee(
			common.TradingFee{
				"taker": 0.0025,
				"maker": 0.0025,
			},
			common.NewFundingFee(
				map[string]float32{
					"ETH": 0.006,
					"OMG": 0.3,
					"DGD": 0.034,
					"CVC": 6,
					"FUN": 36,
					"MCO": 0.35,
					"GNT": 6,
					"ADX": 2.5,
					"PAY": 1.5,
					"BAT": 11,
				},
				map[string]float32{
					"ETH": 0,
					"OMG": 0,
					"DGD": 0,
					"CVC": 0,
					"FUN": 0,
					"MCO": 0,
					"GNT": 0,
					"ADX": 0,
					"PAY": 0,
					"BAT": 0,
				},
			),
		),
	}
}
