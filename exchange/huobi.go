package exchange

import (
	"log"

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
	add, supported := self.addresses[token.ID]
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
	// TODO: update precision and limit
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
