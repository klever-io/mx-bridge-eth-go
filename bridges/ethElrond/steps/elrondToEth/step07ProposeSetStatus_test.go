package elrondToEth

import (
	"context"
	"testing"

	"github.com/ElrondNetwork/elrond-eth-bridge/clients"
	"github.com/ElrondNetwork/elrond-eth-bridge/core"
	bridgeTests "github.com/ElrondNetwork/elrond-eth-bridge/testsCommon/bridge"
	"github.com/stretchr/testify/assert"
)

func TestExecute_ProposeSetStatus(t *testing.T) {
	t.Parallel()
	t.Run("nil batch on GetStoredBatch", func(t *testing.T) {
		t.Parallel()
		bridgeStub := createStubExecutorProposeSetStatus()
		bridgeStub.GetStoredBatchCalled = func() *clients.TransferBatch {
			return nil
		}

		step := proposeSetStatusStep{
			bridge: bridgeStub,
		}

		stepIdentifier := step.Execute(context.Background())
		assert.Equal(t, initialStep, stepIdentifier)
	})

	t.Run("error on WasSetStatusProposedOnElrond", func(t *testing.T) {
		t.Parallel()
		bridgeStub := createStubExecutorProposeSetStatus()
		bridgeStub.WasSetStatusProposedOnElrondCalled = func(ctx context.Context) (bool, error) {
			return false, expectedError
		}

		step := proposeSetStatusStep{
			bridge: bridgeStub,
		}

		stepIdentifier := step.Execute(context.Background())
		assert.Equal(t, initialStep, stepIdentifier)
	})

	t.Run("error on ProposeSetStatusOnElrond", func(t *testing.T) {
		t.Parallel()
		bridgeStub := createStubExecutorProposeSetStatus()
		bridgeStub.ProposeSetStatusOnElrondCalled = func(ctx context.Context) error {
			return expectedError
		}

		step := proposeSetStatusStep{
			bridge: bridgeStub,
		}

		stepIdentifier := step.Execute(context.Background())
		assert.Equal(t, initialStep, stepIdentifier)
	})

	t.Run("should work", func(t *testing.T) {
		t.Parallel()
		t.Run("if SetStatus was proposed it should go to SigningProposedSetStatusOnElrond", func(t *testing.T) {
			t.Parallel()
			bridgeStub := createStubExecutorProposeSetStatus()
			bridgeStub.WasSetStatusProposedOnElrondCalled = func(ctx context.Context) (bool, error) {
				return true, nil
			}

			step := proposeSetStatusStep{
				bridge: bridgeStub,
			}

			assert.False(t, step.IsInterfaceNil())
			expectedStep := core.StepIdentifier(SigningProposedSetStatusOnElrond)
			stepIdentifier := step.Execute(context.Background())
			assert.Equal(t, expectedStep, stepIdentifier)

		})
		t.Run("if SetStatus was not proposed", func(t *testing.T) {
			t.Parallel()
			t.Run("if not leader, should stay in current step", func(t *testing.T) {
				t.Parallel()
				bridgeStub := createStubExecutorProposeSetStatus()
				bridgeStub.MyTurnAsLeaderCalled = func() bool {
					return false
				}
				step := proposeSetStatusStep{
					bridge: bridgeStub,
				}

				stepIdentifier := step.Execute(context.Background())
				assert.Equal(t, step.Identifier(), stepIdentifier)

			})
			t.Run("if leader, should go to SigningProposedTransferOnElrond", func(t *testing.T) {
				t.Parallel()
				bridgeStub := createStubExecutorProposeSetStatus()

				step := proposeSetStatusStep{
					bridge: bridgeStub,
				}

				expectedStep := core.StepIdentifier(SigningProposedSetStatusOnElrond)
				stepIdentifier := step.Execute(context.Background())
				assert.Equal(t, expectedStep, stepIdentifier)

			})
		})

	})
}

func createStubExecutorProposeSetStatus() *bridgeTests.BridgeExecutorStub {
	stub := bridgeTests.NewBridgeExecutorStub()
	stub.GetStoredBatchCalled = func() *clients.TransferBatch {
		return testBatch
	}
	stub.WasSetStatusProposedOnElrondCalled = func(ctx context.Context) (bool, error) {
		return false, nil
	}
	stub.MyTurnAsLeaderCalled = func() bool {
		return true
	}
	stub.ProposeSetStatusOnElrondCalled = func(ctx context.Context) error {
		return nil
	}
	return stub
}
