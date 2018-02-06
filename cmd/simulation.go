package main

import (
	"log"

	"github.com/KyberNetwork/reserve-data/common"
	"github.com/KyberNetwork/reserve-data/data/fetcher/http_runner"
	"github.com/KyberNetwork/reserve-data/data/storage"
	"github.com/KyberNetwork/reserve-data/signer"
	ethereum "github.com/ethereum/go-ethereum/common"
)

func GetConfigForSimulation() *Config {
	settingPath := "/go/src/github.com/KyberNetwork/reserve-data/cmd/shared/deployment_dev.json"
	addressConfig, err := common.GetAddressConfigFromFile(settingPath)
	if err != nil {
		log.Fatalf("Config file %s is not found. Error: %s", settingPath, err)
	}
	wrapperAddr := ethereum.HexToAddress(addressConfig.Wrapper)
	reserveAddr := ethereum.HexToAddress(addressConfig.Reserve)
	pricingAddr := ethereum.HexToAddress(addressConfig.Pricing)
	burnerAddr := ethereum.HexToAddress(addressConfig.FeeBurner)
	networkAddr := ethereum.HexToAddress(addressConfig.Network)

	common.SupportedTokens = map[string]common.Token{}
	tokens := []common.Token{}
	for id, t := range addressConfig.Tokens {
		tok := common.Token{
			id, t.Address, t.Decimals,
		}
		common.SupportedTokens[id] = tok
		tokens = append(tokens, tok)
	}

	// storage := storage.NewRamStorage()
	// metricStorage := metric.NewRamMetricStorage()
	storage, err := storage.NewBoltStorage("/go/src/github.com/KyberNetwork/reserve-data/cmd/core.db")
	if err != nil {
		panic(err)
	}
	fetcherRunner := http_runner.NewHttpRunner(8001)

	fileSigner, depositSigner := signer.NewFileSigner("/go/src/github.com/KyberNetwork/reserve-data/cmd/config.json")

	exchangePool := NewSimulationExchangePool(
		addressConfig, fileSigner, storage,
	)

	// endpoint := "http://localhost:8545"
	// endpoint := "https://kovan.infura.io"
	// endpoint := "https://kovan.kyber.network"
	endpoint := "http://blockchain:8545"

	return &Config{
		ActivityStorage:         storage,
		DataStorage:             storage,
		FetcherStorage:          storage,
		MetricStorage:           storage,
		FetcherRunner:           fetcherRunner,
		FetcherExchanges:        exchangePool.FetcherExchanges(),
		Exchanges:               exchangePool.CoreExchanges(),
		BlockchainSigner:        fileSigner,
		DepositSigner:           depositSigner,
		EthereumEndpoint:        endpoint,
		BackupEthereumEndpoints: []string{},
		SupportedTokens:         tokens,
		WrapperAddress:          wrapperAddr,
		PricingAddress:          pricingAddr,
		ReserveAddress:          reserveAddr,
		FeeBurnerAddress:        burnerAddr,
		NetworkAddress:          networkAddr,
	}
}
