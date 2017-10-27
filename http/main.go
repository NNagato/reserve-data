package main

import (
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/KyberNetwork/reserve-data/blockchain"
	"github.com/KyberNetwork/reserve-data/common"
	"github.com/KyberNetwork/reserve-data/data"
	"github.com/KyberNetwork/reserve-data/data/fetcher"
	"github.com/KyberNetwork/reserve-data/data/storage"
	"github.com/KyberNetwork/reserve-data/exchange"
	"github.com/KyberNetwork/reserve-data/exchange/signer"
	ethereum "github.com/ethereum/go-ethereum/common"
)

func main() {
	numCPU := runtime.NumCPU()
	runtime.GOMAXPROCS(numCPU)

	storage := storage.NewRamStorage()
	fetcher := fetcher.NewFetcher(
		storage, 3*time.Second, 2*time.Second,
		ethereum.HexToAddress("0x7811f3b0505f621bac23cc0ad01bc8ccb68bbfdb"),
	)

	fileSigner := signer.NewFileSigner("config.json")

	fakeEndpoint := "http://127.0.0.1:8000"
	if len(os.Args) > 1 {
		fakeEndpoint = os.Args[1]
	}
	fetcher.AddExchange(exchange.NewLiqui(
		fileSigner,
		// exchange.NewRealLiquiEndpoint(),
		exchange.NewSimulatedLiquiEndpoint(fakeEndpoint),
	))
	// fetcher.AddExchange(exchange.NewBinance())
	// fetcher.AddExchange(exchange.NewBittrex())
	// fetcher.AddExchange(exchange.NewBitfinex())

	bc, err := blockchain.NewBlockchain(
		ethereum.HexToAddress("0x96aa24f61f16c28385e0a1c2ffa60a3518ded3ee"),
	)
	bc.AddToken(common.MustGetToken("ETH"))
	bc.AddToken(common.MustGetToken("OMG"))
	bc.AddToken(common.MustGetToken("DGD"))
	bc.AddToken(common.MustGetToken("CVC"))
	bc.AddToken(common.MustGetToken("MCO"))
	bc.AddToken(common.MustGetToken("GNT"))
	bc.AddToken(common.MustGetToken("ADX"))
	bc.AddToken(common.MustGetToken("EOS"))
	bc.AddToken(common.MustGetToken("PAY"))
	bc.AddToken(common.MustGetToken("BAT"))
	bc.AddToken(common.MustGetToken("KNC"))
	if err != nil {
		fmt.Printf("Can't connect to infura: %s\n", err)
	} else {
		fetcher.SetBlockchain(bc)
		app := data.NewReserveData(
			storage,
			fetcher,
		)
		app.Run()
		server := NewHTTPServer(app, ":8000")
		server.Run()
	}
}