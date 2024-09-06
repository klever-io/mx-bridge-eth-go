package main

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	ethereumClient "github.com/multiversx/mx-bridge-eth-go/clients/ethereum"
	"github.com/multiversx/mx-bridge-eth-go/clients/ethereum/contract"
	"github.com/multiversx/mx-bridge-eth-go/clients/ethereum/wrappers"
	"github.com/multiversx/mx-bridge-eth-go/clients/gasManagement"
	"github.com/multiversx/mx-bridge-eth-go/clients/gasManagement/factory"
	"github.com/multiversx/mx-bridge-eth-go/clients/multiversx"
	"github.com/multiversx/mx-bridge-eth-go/cmd/migration/disabled"
	"github.com/multiversx/mx-bridge-eth-go/config"
	"github.com/multiversx/mx-bridge-eth-go/core"
	"github.com/multiversx/mx-bridge-eth-go/executors/ethereum"
	chainCore "github.com/multiversx/mx-chain-core-go/core"
	logger "github.com/multiversx/mx-chain-logger-go"
	"github.com/multiversx/mx-sdk-go/blockchain"
	sdkCore "github.com/multiversx/mx-sdk-go/core"
	"github.com/multiversx/mx-sdk-go/data"
	"github.com/urfave/cli"
)

const (
	filePathPlaceholder  = "[path]"
	signMode             = "sign"
	executeMode          = "execute"
	configPath           = "config"
	timestampPlaceholder = "[timestamp]"
	publicKeyPlaceholder = "[public-key]"
)

var log = logger.GetOrCreate("main")

type internalComponents struct {
	batch         *ethereum.BatchInfo
	cryptoHandler ethereumClient.CryptoHandler
	ethClient     *ethclient.Client
}

func main() {
	app := cli.NewApp()
	app.Name = "Funds migration CLI tool"
	app.Usage = "This is the entry point for the migration CLI tool"
	app.Flags = getFlags()
	app.Authors = []cli.Author{
		{
			Name:  "The MultiversX Team",
			Email: "contact@multiversx.com",
		},
	}

	app.Action = func(c *cli.Context) error {
		return execute(c)
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}

	log.Info("process finished successfully")
}

func execute(ctx *cli.Context) error {
	flagsConfig := getFlagsConfig(ctx)

	err := logger.SetLogLevel(flagsConfig.LogLevel)
	if err != nil {
		return err
	}

	cfg, err := loadConfig(flagsConfig.ConfigurationFile)
	if err != nil {
		return err
	}

	log.Info("starting migration help tool", "pid", os.Getpid())

	operationMode := strings.ToLower(ctx.GlobalString(mode.Name))
	switch operationMode {
	case signMode:

		_, err = generateAndSign(ctx, cfg)
		return err
	case executeMode:
		return executeTransfer(ctx, cfg)
	}

	return fmt.Errorf("unknown execution mode: %s", operationMode)
}

func generateAndSign(ctx *cli.Context, cfg config.MigrationToolConfig) (*internalComponents, error) {
	argsProxy := blockchain.ArgsProxy{
		ProxyURL:            cfg.MultiversX.NetworkAddress,
		SameScState:         false,
		ShouldBeSynced:      false,
		FinalityCheck:       cfg.MultiversX.Proxy.FinalityCheck,
		AllowedDeltaToFinal: cfg.MultiversX.Proxy.MaxNoncesDelta,
		CacheExpirationTime: time.Second * time.Duration(cfg.MultiversX.Proxy.CacherExpirationSeconds),
		EntityType:          sdkCore.RestAPIEntityType(cfg.MultiversX.Proxy.RestAPIEntityType),
	}
	proxy, err := blockchain.NewProxy(argsProxy)
	if err != nil {
		return nil, err
	}

	dummyAddress := data.NewAddressFromBytes(bytes.Repeat([]byte{0x1}, 32))
	multisigAddress, err := data.NewAddressFromBech32String(cfg.MultiversX.MultisigContractAddress)
	if err != nil {
		return nil, err
	}

	safeAddress, err := data.NewAddressFromBech32String(cfg.MultiversX.SafeContractAddress)
	if err != nil {
		return nil, err
	}

	argsMXClientDataGetter := multiversx.ArgsMXClientDataGetter{
		MultisigContractAddress: multisigAddress,
		SafeContractAddress:     safeAddress,
		RelayerAddress:          dummyAddress,
		Proxy:                   proxy,
		Log:                     log,
	}
	mxDataGetter, err := multiversx.NewMXClientDataGetter(argsMXClientDataGetter)
	if err != nil {
		return nil, err
	}

	ethClient, err := ethclient.Dial(cfg.Eth.NetworkAddress)
	if err != nil {
		return nil, err
	}

	argsContractsHolder := ethereumClient.ArgsErc20SafeContractsHolder{
		EthClient:              ethClient,
		EthClientStatusHandler: &disabled.StatusHandler{},
	}
	erc20ContractsHolder, err := ethereumClient.NewErc20SafeContractsHolder(argsContractsHolder)
	if err != nil {
		return nil, err
	}

	safeEthAddress := common.HexToAddress(cfg.Eth.SafeContractAddress)
	safeInstance, err := contract.NewERC20Safe(safeEthAddress, ethClient)
	if err != nil {
		return nil, err
	}

	argsCreator := ethereum.ArgsMigrationBatchCreator{
		MvxDataGetter:        mxDataGetter,
		Erc20ContractsHolder: erc20ContractsHolder,
		SafeContractAddress:  safeEthAddress,
		SafeContractWrapper:  safeInstance,
		Logger:               log,
	}

	creator, err := ethereum.NewMigrationBatchCreator(argsCreator)
	if err != nil {
		return nil, err
	}

	newSafeAddressString := ctx.GlobalString(newSafeAddress.Name)
	if len(newSafeAddressString) == 0 {
		return nil, fmt.Errorf("invalid new safe address for Ethereum")
	}
	newSafeAddressValue := common.HexToAddress(ctx.GlobalString(newSafeAddress.Name))

	batchInfo, err := creator.CreateBatchInfo(context.Background(), newSafeAddressValue)
	if err != nil {
		return nil, err
	}

	val, err := json.MarshalIndent(batchInfo, "", "  ")
	if err != nil {
		return nil, err
	}

	cryptoHandler, err := ethereumClient.NewCryptoHandler(cfg.Eth.PrivateKeyFile)
	if err != nil {
		return nil, err
	}

	log.Info("signing batch", "message hash", batchInfo.MessageHash.String(),
		"public key", cryptoHandler.GetAddress().String())

	signature, err := cryptoHandler.Sign(batchInfo.MessageHash)
	if err != nil {
		return nil, err
	}

	log.Info("Migration .json file contents: \n" + string(val))

	jsonFilename := ctx.GlobalString(migrationJsonFile.Name)
	jsonFilename = applyTimestamp(jsonFilename)
	err = os.WriteFile(jsonFilename, val, os.ModePerm)
	if err != nil {
		return nil, err
	}

	sigInfo := &ethereum.SignatureInfo{
		Address:     cryptoHandler.GetAddress().String(),
		MessageHash: batchInfo.MessageHash.String(),
		Signature:   hex.EncodeToString(signature),
	}

	sigFilename := ctx.GlobalString(signatureJsonFile.Name)
	sigFilename = applyTimestamp(sigFilename)
	sigFilename = applyPublicKey(sigFilename, sigInfo.Address)
	val, err = json.MarshalIndent(sigInfo, "", "  ")
	if err != nil {
		return nil, err
	}

	log.Info("Signature .json file contents: \n" + string(val))

	err = os.WriteFile(sigFilename, val, os.ModePerm)
	if err != nil {
		return nil, err
	}

	return &internalComponents{
		batch:         batchInfo,
		cryptoHandler: cryptoHandler,
		ethClient:     ethClient,
	}, nil
}

func executeTransfer(ctx *cli.Context, cfg config.MigrationToolConfig) error {
	components, err := generateAndSign(ctx, cfg)
	if err != nil {
		return err
	}

	bridgeEthAddress := common.HexToAddress(cfg.Eth.MultisigContractAddress)
	multiSigInstance, err := contract.NewBridge(bridgeEthAddress, components.ethClient)
	if err != nil {
		return err
	}

	safeEthAddress := common.HexToAddress(cfg.Eth.SafeContractAddress)
	safeInstance, err := contract.NewERC20Safe(safeEthAddress, components.ethClient)
	if err != nil {
		return err
	}

	argsClientWrapper := wrappers.ArgsEthereumChainWrapper{
		StatusHandler:    &disabled.StatusHandler{},
		MultiSigContract: multiSigInstance,
		SafeContract:     safeInstance,
		BlockchainClient: components.ethClient,
	}
	ethereumChainWrapper, err := wrappers.NewEthereumChainWrapper(argsClientWrapper)
	if err != nil {
		return err
	}

	gasStationConfig := cfg.Eth.GasStation
	argsGasStation := gasManagement.ArgsGasStation{
		RequestURL:             gasStationConfig.URL,
		RequestPollingInterval: time.Duration(gasStationConfig.PollingIntervalInSeconds) * time.Second,
		RequestRetryDelay:      time.Duration(gasStationConfig.RequestRetryDelayInSeconds) * time.Second,
		MaximumFetchRetries:    gasStationConfig.MaxFetchRetries,
		RequestTime:            time.Duration(gasStationConfig.RequestTimeInSeconds) * time.Second,
		MaximumGasPrice:        gasStationConfig.MaximumAllowedGasPrice,
		GasPriceSelector:       core.EthGasPriceSelector(gasStationConfig.GasPriceSelector),
		GasPriceMultiplier:     gasStationConfig.GasPriceMultiplier,
	}
	gs, err := factory.CreateGasStation(argsGasStation, gasStationConfig.Enabled)
	if err != nil {
		return err
	}

	args := ethereum.ArgsMigrationBatchExecutor{
		EthereumChainWrapper:    ethereumChainWrapper,
		CryptoHandler:           components.cryptoHandler,
		Batch:                   *components.batch,
		Signatures:              ethereum.LoadAllSignatures(log, configPath),
		Logger:                  log,
		GasHandler:              gs,
		TransferGasLimitBase:    cfg.Eth.GasLimitBase,
		TransferGasLimitForEach: cfg.Eth.GasLimitForEach,
	}

	executor, err := ethereum.NewMigrationBatchExecutor(args)
	if err != nil {
		return err
	}

	return executor.ExecuteTransfer(context.Background())
}

func loadConfig(filepath string) (config.MigrationToolConfig, error) {
	cfg := config.MigrationToolConfig{}
	err := chainCore.LoadTomlFile(&cfg, filepath)
	if err != nil {
		return config.MigrationToolConfig{}, err
	}

	return cfg, nil
}

func applyTimestamp(input string) string {
	actualTimestamp := time.Now().Format("2006-01-02T15-04-05")
	actualTimestamp = strings.Replace(actualTimestamp, "T", "-", 1)

	return strings.Replace(input, timestampPlaceholder, actualTimestamp, 1)
}

func applyPublicKey(input string, publickey string) string {
	return strings.Replace(input, publicKeyPlaceholder, publickey, 1)
}
