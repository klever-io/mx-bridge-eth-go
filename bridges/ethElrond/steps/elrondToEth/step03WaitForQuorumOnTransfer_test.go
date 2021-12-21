package elrondToEth

import (
	"context"
	"testing"

	"github.com/ElrondNetwork/elrond-eth-bridge/core"
	bridgeTests "github.com/ElrondNetwork/elrond-eth-bridge/testsCommon/bridge"
	"github.com/stretchr/testify/assert"
)

func TestExecute_WaitForQuorumOnTransfer(t *testing.T) {
	t.Parallel()

	t.Run("error on ProcessQuorumReachedOnEthereum", func(t *testing.T) {
		t.Parallel()
		bridgeStub := createStubExecutorWaitForQuorumOnTransfer()
		bridgeStub.ProcessQuorumReachedOnEthereumCalled = func(ctx context.Context) (bool, error) {
			return false, expectedError
		}

		step := waitForQuorumOnTransferStep{
			bridge: bridgeStub,
		}

		stepIdentifier := step.Execute(context.Background())
		assert.Equal(t, initialStep, stepIdentifier)
	})

	t.Run("max retries reached", func(t *testing.T) {
		t.Parallel()
		bridgeStub := createStubExecutorWaitForQuorumOnTransfer()
		bridgeStub.ProcessMaxRetriesOnEthereumCalled = func() bool {
			return true
		}

		step := waitForQuorumOnTransferStep{
			bridge: bridgeStub,
		}

		stepIdentifier := step.Execute(context.Background())
		assert.Equal(t, initialStep, stepIdentifier)
	})

	t.Run("quorum not reached", func(t *testing.T) {
		t.Parallel()
		bridgeStub := createStubExecutorWaitForQuorumOnTransfer()
		bridgeStub.ProcessQuorumReachedOnEthereumCalled = func(ctx context.Context) (bool, error) {
			return false, nil
		}

		step := waitForQuorumOnTransferStep{
			bridge: bridgeStub,
		}

		assert.False(t, step.IsInterfaceNil())

		expectedStepIdentifier := step.Identifier()
		stepIdentifier := step.Execute(context.Background())
		assert.Equal(t, expectedStepIdentifier, stepIdentifier)
	})

	t.Run("quorum reached", func(t *testing.T) {
		t.Parallel()
		bridgeStub := createStubExecutorWaitForQuorumOnTransfer()
		bridgeStub.ProcessQuorumReachedOnEthereumCalled = func(ctx context.Context) (bool, error) {
			return true, nil
		}

		step := waitForQuorumOnTransferStep{
			bridge: bridgeStub,
		}

		assert.False(t, step.IsInterfaceNil())

		expectedStepIdentifier := core.StepIdentifier(PerformingTransfer)
		stepIdentifier := step.Execute(context.Background())
		assert.Equal(t, expectedStepIdentifier, stepIdentifier)
	})
}

func createStubExecutorWaitForQuorumOnTransfer() *bridgeTests.BridgeExecutorStub {
	stub := bridgeTests.NewBridgeExecutorStub()
	stub.ProcessMaxRetriesOnEthereumCalled = func() bool {
		return false
	}
	return stub
}
