//go:build slow

// To run these slow tests, simply add the slow tag on the go test command. Also, provide a chain simulator instance on the 8085 port
// example: go test -tags slow

package slowTests

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"os/signal"
	"syscall"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/multiversx/mx-bridge-eth-go/core/batchProcessor"
	"github.com/multiversx/mx-bridge-eth-go/integrationTests/mock"
	logger "github.com/multiversx/mx-chain-logger-go"
	"github.com/multiversx/mx-sdk-go/data"
	"github.com/stretchr/testify/require"
)

const (
	timeout = time.Minute * 15
)

func TestRelayersShouldExecuteTransfers(t *testing.T) {
	t.Run("ETH->MVX and back, ethNative = true, ethMintBurn = false, mvxNative = false, mvxMintBurn = true", func(t *testing.T) {
		args := argSimulatedSetup{
			mvxIsMintBurn:        true,
			mvxIsNative:          false,
			ethIsMintBurn:        false,
			ethIsNative:          true,
			transferBackAndForth: true,
		}
		testRelayersShouldExecuteTransfersEthToMVX(t, args)
	})
	t.Run("MVX->ETH, ethNative = false, ethMintBurn = true, mvxNative = true, mvxMintBurn = false", func(t *testing.T) {
		args := argSimulatedSetup{
			mvxIsMintBurn:        false,
			mvxIsNative:          true,
			ethIsMintBurn:        true,
			ethIsNative:          false,
			transferBackAndForth: true,
		}
		testRelayersShouldExecuteTransfersMVXToETH(t, args)
	})
	t.Run("ETH->MVX with SC call that works, ethNative = true, ethMintBurn = false, mvxNative = false, mvxMintBurn = true", func(t *testing.T) {
		args := argSimulatedSetup{
			mvxIsMintBurn:        true,
			mvxIsNative:          false,
			ethIsMintBurn:        false,
			ethIsNative:          true,
			ethSCCallMethod:      "callPayable",
			ethSCCallGasLimit:    50000000,
			ethSCCallArguments:   nil,
			transferBackAndForth: false,
		}
		testRelayersShouldExecuteTransfersEthToMVX(t, args)
	})
}

func testRelayersShouldExecuteTransfersEthToMVX(t *testing.T, argsSimulatedSetup argSimulatedSetup) {
	defer func() {
		r := recover()
		if r != nil {
			require.Fail(t, "should have not panicked")
		}
	}()

	argsSimulatedSetup.t = t
	testSetup := prepareSimulatedSetup(argsSimulatedSetup)
	defer testSetup.close()

	testSetup.checkESDTBalance(testSetup.mvxReceiverAddress, testSetup.mvxUniversalToken, "0", true)

	testSetup.createBatch(batchProcessor.ToMultiversX)

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGINT, syscall.SIGTERM)
	ethToMVXDone := false
	mvxToETHDone := false

	safeAddr, err := data.NewAddressFromBech32String(testSetup.mvxSafeAddress)
	require.NoError(t, err)

	// send half of the amount back to ETH
	valueToSendFromMVX := big.NewInt(0).Div(mintAmount, big.NewInt(2))
	initialSafeValue, err := testSetup.mvxChainSimulator.GetESDTBalance(testSetup.testContext, safeAddr, testSetup.mvxChainSpecificToken)
	require.NoError(t, err)
	initialSafeValueInt, _ := big.NewInt(0).SetString(initialSafeValue, 10)
	expectedFinalValueOnMVXSafe := initialSafeValueInt.Add(initialSafeValueInt, feeInt)
	expectedFinalValueOnETH := big.NewInt(0).Sub(valueToSendFromMVX, feeInt)
	for {
		select {
		case <-interrupt:
			require.Fail(t, "signal interrupted")
			return
		case <-time.After(timeout):
			require.Fail(t, "time out")
			return
		default:
			receiverToCheckBalance := testSetup.mvxReceiverAddress
			if len(testSetup.ethSCCallMethod) > 0 {
				receiverToCheckBalance = testSetup.mvxTestCallerAddress
			}

			isTransferDoneFromETH := testSetup.checkESDTBalance(receiverToCheckBalance, testSetup.mvxUniversalToken, mintAmount.String(), false)
			if !ethToMVXDone && isTransferDoneFromETH {
				ethToMVXDone = true

				if argsSimulatedSetup.transferBackAndForth {
					log.Info("ETH->MvX transfer finished, now sending back to ETH...")

					testSetup.sendMVXToEthTransaction(valueToSendFromMVX.Bytes())
				} else {
					log.Info("ETH->MvX transfers done")
					return
				}
			}

			isTransferDoneFromMVX := testSetup.checkETHStatus(testSetup.ethOwnerAddress, expectedFinalValueOnETH.Uint64())
			safeSavedFee := testSetup.checkESDTBalance(safeAddr, testSetup.mvxChainSpecificToken, expectedFinalValueOnMVXSafe.String(), false)
			if !mvxToETHDone && isTransferDoneFromMVX && safeSavedFee {
				mvxToETHDone = true
			}

			if ethToMVXDone && mvxToETHDone {
				log.Info("MvX<->ETH transfers done")
				return
			}

			// commit blocks in order to execute incoming txs from relayers
			testSetup.simulatedETHChain.Commit()

			testSetup.mvxChainSimulator.GenerateBlocks(testSetup.testContext, 1)

		case <-interrupt:
			require.Fail(t, "signal interrupted")
			return
		case <-time.After(timeout):
			require.Fail(t, "time out")
			return
		}
	}
}

func testRelayersShouldExecuteTransfersMVXToETH(t *testing.T, argsSimulatedSetup argSimulatedSetup) {
	defer func() {
		r := recover()
		if r != nil {
			require.Fail(t, "should have not panicked")
		}
	}()

	argsSimulatedSetup.t = t
	testSetup := prepareSimulatedSetup(argsSimulatedSetup)
	defer testSetup.close()

	testSetup.checkESDTBalance(testSetup.mvxReceiverAddress, testSetup.mvxUniversalToken, "0", true)

	safeAddr, err := data.NewAddressFromBech32String(testSetup.mvxSafeAddress)
	require.NoError(t, err)

	initialSafeValue, err := testSetup.mvxChainSimulator.GetESDTBalance(testSetup.testContext, safeAddr, testSetup.mvxChainSpecificToken)
	require.NoError(t, err)

	testSetup.createBatch(batchProcessor.FromMultiversX)

	// wait for signal interrupt or time out
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGINT, syscall.SIGTERM)

	// send half of the amount back to ETH
	valueSentFromETH := big.NewInt(0).Div(mintAmount, big.NewInt(2))
	initialSafeValueInt, _ := big.NewInt(0).SetString(initialSafeValue, 10)
	expectedFinalValueOnMVXSafe := initialSafeValueInt.Add(initialSafeValueInt, valueSentFromETH)
	expectedFinalValueOnETH := big.NewInt(0).Sub(valueSentFromETH, feeInt)
	expectedFinalValueOnETH = expectedFinalValueOnETH.Mul(expectedFinalValueOnETH, big.NewInt(1000000))
	for {
		select {
		case <-interrupt:
			require.Fail(t, "signal interrupted")
			return
		case <-time.After(timeout):
			require.Fail(t, "time out")
			return
		default:
			isTransferDoneFromMVX := testSetup.checkETHStatus(testSetup.ethOwnerAddress, expectedFinalValueOnETH.Uint64())
			safeSavedFunds := testSetup.checkESDTBalance(safeAddr, testSetup.mvxChainSpecificToken, expectedFinalValueOnMVXSafe.String(), false)
			if isTransferDoneFromMVX && safeSavedFunds {
				log.Info("MVX->ETH transfer finished")

				return
			}

			// commit blocks in order to execute incoming txs from relayers
			testSetup.simulatedETHChain.Commit()

			testSetup.mvxChainSimulator.GenerateBlocks(testSetup.testContext, 1)
		}
	}
}

func TestRelayersShouldNotExecuteTransfers(t *testing.T) {
	t.Run("ETH->MVX, ethNative = true, ethMintBurn = false, mvxNative = true, mvxMintBurn = false", func(t *testing.T) {
		args := argSimulatedSetup{
			mvxIsMintBurn: false,
			mvxIsNative:   true,
			ethIsMintBurn: false,
			ethIsNative:   true,
		}
		expectedStringInLogs := "error = invalid setup isNativeOnEthereum = true, isNativeOnMultiversX = true"
		testRelayersShouldNotExecuteTransfers(t, args, expectedStringInLogs, batchProcessor.ToMultiversX)
	})
	t.Run("ETH->MVX, ethNative = true, ethMintBurn = false, mvxNative = true, mvxMintBurn = true", func(t *testing.T) {
		args := argSimulatedSetup{
			mvxIsMintBurn: true,
			mvxIsNative:   true,
			ethIsMintBurn: false,
			ethIsNative:   true,
		}
		expectedStringInLogs := "error = invalid setup isNativeOnEthereum = true, isNativeOnMultiversX = true"
		testRelayersShouldNotExecuteTransfers(t, args, expectedStringInLogs, batchProcessor.ToMultiversX)
	})
	t.Run("ETH->MVX, ethNative = true, ethMintBurn = true, mvxNative = true, mvxMintBurn = false", func(t *testing.T) {
		args := argSimulatedSetup{
			mvxIsMintBurn: false,
			mvxIsNative:   true,
			ethIsMintBurn: true,
			ethIsNative:   true,
		}
		testEthContractsShouldError(t, args)
	})
	t.Run("ETH->MVX, ethNative = true, ethMintBurn = true, mvxNative = true, mvxMintBurn = true", func(t *testing.T) {
		args := argSimulatedSetup{
			mvxIsMintBurn: true,
			mvxIsNative:   true,
			ethIsMintBurn: true,
			ethIsNative:   true,
		}
		testEthContractsShouldError(t, args)
	})
	t.Run("ETH->MVX, ethNative = false, ethMintBurn = true, mvxNative = false, mvxMintBurn = true", func(t *testing.T) {
		args := argSimulatedSetup{
			mvxIsMintBurn: true,
			mvxIsNative:   false,
			ethIsMintBurn: true,
			ethIsNative:   false,
		}
		testEthContractsShouldError(t, args)
	})
	t.Run("MVX->ETH, ethNative = true, ethMintBurn = false, mvxNative = true, mvxMintBurn = false", func(t *testing.T) {
		args := argSimulatedSetup{
			mvxIsMintBurn: false,
			mvxIsNative:   true,
			ethIsMintBurn: false,
			ethIsNative:   true,
		}
		expectedStringInLogs := "error = invalid setup isNativeOnEthereum = true, isNativeOnMultiversX = true"
		testRelayersShouldNotExecuteTransfers(t, args, expectedStringInLogs, batchProcessor.FromMultiversX)
	})
}

func testRelayersShouldNotExecuteTransfers(
	t *testing.T,
	argsSimulatedSetup argSimulatedSetup,
	expectedStringInLogs string,
	direction batchProcessor.Direction,
) {
	defer func() {
		r := recover()
		if r != nil {
			require.Fail(t, "should have not panicked")
		}
	}()

	argsSimulatedSetup.t = t
	testSetup := prepareSimulatedSetup(argsSimulatedSetup)
	defer testSetup.close()

	testSetup.checkESDTBalance(testSetup.mvxReceiverAddress, testSetup.mvxUniversalToken, "0", true)

	testSetup.createBatch(direction)

	// start a mocked log observer that is looking for a specific relayer error
	chanCnt := 0
	mockLogObserver := mock.NewMockLogObserver(expectedStringInLogs)
	err := logger.AddLogObserver(mockLogObserver, &logger.PlainFormatter{})
	require.NoError(t, err)
	defer func() {
		require.NoError(t, logger.RemoveLogObserver(mockLogObserver))
	}()

	numOfTimesToRepeatErrorForRelayer := 10
	numOfErrorsToWait := numOfTimesToRepeatErrorForRelayer * numRelayers

	// wait for signal interrupt or time out
	roundDuration := time.Second
	roundTimer := time.NewTimer(roundDuration)
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGINT, syscall.SIGTERM)

	for {
		roundTimer.Reset(roundDuration)
		select {
		case <-interrupt:
			require.Fail(t, "signal interrupted")
			return
		case <-time.After(timeout):
			require.Fail(t, "time out")
			return
		case <-mockLogObserver.LogFoundChan():
			chanCnt++
			if chanCnt >= numOfErrorsToWait {
				testSetup.checkESDTBalance(testSetup.mvxReceiverAddress, testSetup.mvxUniversalToken, "0", true)

				log.Info(fmt.Sprintf("test passed, relayers are stuck, expected string `%s` found in all relayers' logs for %d times", expectedStringInLogs, numOfErrorsToWait))

				return
			}
		case <-roundTimer.C:
			// commit blocks
			testSetup.simulatedETHChain.Commit()

			testSetup.mvxChainSimulator.GenerateBlocks(testSetup.testContext, 1)
		}
	}
}

func testEthContractsShouldError(t *testing.T, argsSimulatedSetup argSimulatedSetup) {
	defer func() {
		r := recover()
		if r != nil {
			require.Fail(t, "should have not panicked")
		}
	}()

	testSetup := &simulatedSetup{}
	testSetup.T = t

	// create a test context
	testSetup.testContext, testSetup.testContextCancel = context.WithCancel(context.Background())

	testSetup.workingDir = t.TempDir()

	testSetup.generateKeys()

	receiverKeys := generateMvxPrivatePublicKey(t)
	mvxReceiverAddress, err := data.NewAddressFromBech32String(receiverKeys.pk)
	require.NoError(t, err)

	testSetup.ethOwnerAddress = crypto.PubkeyToAddress(ethOwnerSK.PublicKey)
	ethDepositorAddr := crypto.PubkeyToAddress(ethDepositorSK.PublicKey)

	// create ethereum simulator
	testSetup.createEthereumSimulatorAndDeployContracts(ethDepositorAddr, argsSimulatedSetup.ethIsMintBurn, argsSimulatedSetup.ethIsNative)

	// add allowance for the sender
	auth, _ := bind.NewKeyedTransactorWithChainID(ethDepositorSK, testSetup.ethChainID)
	tx, err := testSetup.ethGenericTokenContract.Approve(auth, testSetup.ethSafeAddress, mintAmount)
	require.NoError(t, err)
	testSetup.simulatedETHChain.Commit()
	testSetup.checkEthTxResult(tx.Hash())

	// deposit on ETH safe should fail due to bad setup
	auth, _ = bind.NewKeyedTransactorWithChainID(ethDepositorSK, testSetup.ethChainID)
	_, err = testSetup.ethSafeContract.Deposit(auth, testSetup.ethGenericTokenAddress, mintAmount, mvxReceiverAddress.AddressSlice())
	require.Error(t, err)
}
