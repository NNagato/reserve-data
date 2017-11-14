package main

import (
	"log"

	"github.com/KyberNetwork/reserve-data/common"
	corestorage "github.com/KyberNetwork/reserve-data/core/storage"
	"github.com/KyberNetwork/reserve-data/data/fetcher"
	"github.com/KyberNetwork/reserve-data/data/fetcher/http_runner"
	"github.com/KyberNetwork/reserve-data/data/storage"
	"github.com/KyberNetwork/reserve-data/exchange"
	"github.com/KyberNetwork/reserve-data/exchange/liqui"
	"github.com/KyberNetwork/reserve-data/signer"
	ethereum "github.com/ethereum/go-ethereum/common"
)

func GetConfigForSimulation() *Config {
	settingPath := "/go/src/github.com/KyberNetwork/reserve-data/cmd/simulation_setting.json"
	addressConfig, err := common.GetAddressConfigFromFile(settingPath)
	if err != nil {
		log.Fatalf("Config file %s is not found. Error: %s", settingPath, err)
	}
	wrapperAddr := ethereum.HexToAddress(addressConfig.Wrapper)
	reserveAddr := ethereum.HexToAddress(addressConfig.Reserve)

	common.SupportedTokens = map[string]common.Token{}
	tokens := []common.Token{}
	for id, t := range addressConfig.Tokens {
		tok := common.Token{
			id, t.Address, t.Decimals,
		}
		common.SupportedTokens[id] = tok
		tokens = append(tokens, tok)
	}

	// wrapperAddr := ethereum.HexToAddress("0x5aa7b0c53affef857523014ac6ce6c8d30bc68e6")
	// reserveAddr := ethereum.HexToAddress("0x98990ee596d7c383a496f54c9e617ce7d2b3ed46")

	storage, err := storage.NewBoltStorage("/go/src/github.com/KyberNetwork/reserve-data/cmd/core.db")
	if err != nil {
		panic(err)
	}
	fetcherRunner := http_runner.NewHttpRunner(8001)

	fileSigner := signer.NewFileSigner("/go/src/github.com/KyberNetwork/reserve-data/cmd/config.json")

	fetcherExchanges := []fetcher.Exchange{}
	// liqui := exchange.NewRealLiqui(fileSigner)
	liqui := exchange.NewLiqui(liqui.NewSimulatedLiquiEndpoint(fileSigner))

	fetcherExchanges = append(fetcherExchanges, liqui)

	exchanges := []common.Exchange{}
	exchanges = append(exchanges, liqui)

	// endpoint := "http://localhost:8545"
	// endpoint := "https://kovan.kyber.network"
	endpoint := "http://blockchain:8545"

	activityStorage := corestorage.NewRamStorage()
	return &Config{
		ActivityStorage:  activityStorage,
		DataStorage:      storage,
		FetcherStorage:   storage,
		FetcherRunner:    fetcherRunner,
		FetcherExchanges: fetcherExchanges,
		Exchanges:        exchanges,
		BlockchainSigner: fileSigner,
		EthereumEndpoint: endpoint,
		SupportedTokens:  tokens,
		WrapperAddress:   wrapperAddr,
		ReserveAddress:   reserveAddr,
	}
}