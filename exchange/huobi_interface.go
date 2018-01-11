package exchange

import (
	"math/big"

	"github.com/KyberNetwork/reserve-data/common"
	ethereum "github.com/ethereum/go-ethereum/common"
)

type HuobiInterface interface {
	GetDepthOnePair(
		pair common.TokenPair, timepoint uint64) (HuobiDepth, error)

	OpenOrdersForOnePair(
		pair common.TokenPair, timepoint uint64) (HuobiOrder, error)

	GetInfo(timepoint uint64) (HuobiInfo, error)

	GetExchangeInfo() (HuobiExchangeInfo, error)

	Withdraw(
		token common.Token,
		amount *big.Int,
		address ethereum.Address,
		timepoint uint64) (string, error)

	Trade(
		tradeType string,
		base, quote common.Token,
		rate, amount float64,
		timepoint uint64) (HuobiTrade, error)

	CancelOrder(symbol string, id uint64) (HuobiCancel, error)

	DepositHistory(startTime, endTime uint64) (HuobiDeposit, error)

	WithdrawHistory(
		startTime, endTime uint64) (HuobiWithdraw, error)

	OrderStatus(
		symbol string, id uint64, timepoint uint64) (HuobiOrder, error)
}
