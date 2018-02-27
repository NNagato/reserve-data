package fetcher

import (
	"log"
	"strings"
	"time"

	"github.com/KyberNetwork/reserve-data/common"
)

type Fetcher struct {
	storage                Storage
	blockchain             Blockchain
	runner                 FetcherRunner
	ethRate                *common.EthRate
	currentBlock           uint64
	currentBlockUpdateTime uint64
}

func NewFetcher(
	storage Storage,
	runner FetcherRunner) *Fetcher {
	return &Fetcher{
		storage:    storage,
		blockchain: nil,
		runner:     runner,
	}
}

func (self *Fetcher) Stop() error {
	return self.runner.Stop()
}

func (self *Fetcher) SetBlockchain(blockchain Blockchain) {
	self.blockchain = blockchain
	self.FetchCurrentBlock(common.GetTimepoint())
}

func (self *Fetcher) RunGetEthRate() {
	tick := time.NewTicker(1 * time.Hour)
	go func() {
		for {
			self.ethRate.UpdateEthRate()
			<-tick.C
		}
	}()
}

func (self *Fetcher) Run() error {
	log.Printf("Fetcher runner is starting...")
	self.runner.Start()
	go self.RunGetEthRate()
	go self.RunBlockAndLogFetcher()
	log.Printf("Fetcher runner is running...")
	return nil
}

func (self *Fetcher) RunBlockAndLogFetcher() {
	for {
		log.Printf("waiting for signal from block channel")
		t := <-self.runner.GetBlockTicker()
		log.Printf("got signal in block channel with timestamp %d", common.TimeToTimepoint(t))
		timepoint := common.TimeToTimepoint(t)
		self.FetchCurrentBlock(timepoint)
		log.Printf("fetched block from blockchain")
		lastBlock, err := self.storage.LastBlock()
		if err == nil {
			nextBlock := self.FetchLogs(lastBlock+1, timepoint)
			self.storage.UpdateLogBlock(nextBlock, timepoint)
			log.Printf("nextBlock: %d", nextBlock)
		} else {
			log.Printf("failed to get last fetched log block, err: %+v", err)
		}
	}
}

// return block number that we just fetched the logs
func (self *Fetcher) FetchLogs(fromBlock uint64, timepoint uint64) uint64 {
	logs, err := self.blockchain.GetLogs(fromBlock, timepoint, self.ethRate.GetEthRate())
	if err != nil {
		log.Printf("fetching logs data from block %d failed, error: %v", fromBlock, err)
		if fromBlock == 0 {
			return 0
		} else {
			return fromBlock - 1
		}
	} else {
		if len(logs) > 0 {
			for _, l := range logs {
				log.Printf("blockno: %d - %d", l.BlockNumber, l.TransactionIndex)
				err = self.storage.StoreTradeLog(l, timepoint)
				if err != nil {
					log.Printf("storing trade log failed, abort storing process and return latest stored log block number, err: %+v", err)
					return l.BlockNumber
				} else {
					self.aggregateTradeLog(l)
				}
			}
			return logs[len(logs)-1].BlockNumber
		} else {
			return fromBlock - 1
		}
	}
}

func (self *Fetcher) aggregateTradeLog(trade common.TradeLog) (err error) {
	walletFeeKey := strings.Join([]string{trade.ReserveAddress.String(), trade.WalletAddress.String()}, "_")
	updates := []struct {
		metric     string
		tradeStats common.TradeStats
	}{
		{
			"assets_volume",
			common.TradeStats{
				strings.ToLower(trade.SrcAddress.String()):  trade.SrcAmount,
				strings.ToLower(trade.DestAddress.String()): trade.DestAmount,
			},
		},
		{
			"burn_fee",
			common.TradeStats{
				strings.ToLower(trade.ReserveAddress.String()): trade.BurnFee,
			},
		},
		{
			"wallet_fee",
			common.TradeStats{
				walletFeeKey: trade.WalletFee,
			},
		},
		{
			"user_volume",
			common.TradeStats{
				strings.ToLower(trade.UserAddress.String()): trade.FiatAmount,
			},
		},
	}
	for _, update := range updates {
		for _, freq := range []string{"M", "H", "D"} {
			err = self.storage.SetTradeStats(update.metric, freq, trade.Timestamp, update.tradeStats)
			if err != nil {
				return
			}
		}
	}
	return
}

func (self *Fetcher) FetchCurrentBlock(timepoint uint64) {
	block, err := self.blockchain.CurrentBlock()
	if err != nil {
		log.Printf("Fetching current block failed: %v. Ignored.", err)
	} else {
		// update currentBlockUpdateTime first to avoid race condition
		// where fetcher is trying to fetch new rate
		self.currentBlockUpdateTime = common.GetTimepoint()
		self.currentBlock = block
	}
}
