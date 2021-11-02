package steps

import (
	"context"
	"errors"
	"testing"

	"github.com/ElrondNetwork/elrond-eth-bridge/core"
	"github.com/ElrondNetwork/elrond-eth-bridge/ethToElrond"
	"github.com/ElrondNetwork/elrond-eth-bridge/testsCommon/stateMachine"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFlowAsLeaderProposeTransferOnDestinationErrorsWithStubChecking(t *testing.T) {
	t.Parallel()

	bem := stateMachine.NewBridgeExecutorMock()
	setAllDecisionHandlersToTrue(bem)
	bem.ProposeTransferOnDestinationCalled = func(ctx context.Context) error {
		return errors.New("expected error")
	}

	steps, err := CreateSteps(bem)
	require.Nil(t, err)
	smm := stateMachine.NewStateMachineMock(steps, ethToElrond.GettingPending)
	err = smm.Initialize()
	require.Nil(t, err)

	numSteps := 6
	for i := 0; i < numSteps; i++ {
		err = smm.ExecuteOneStep()
		require.Nil(t, err)
	}

	expectedSteps := []core.StepIdentifier{
		ethToElrond.GettingPending,
		ethToElrond.ProposingTransfer,
		ethToElrond.ProposingSetStatus,
		ethToElrond.WaitingSignaturesForProposeSetStatus,
		ethToElrond.ExecutingSetStatus,
		ethToElrond.GettingPending,
	}

	assert.Equal(t, expectedSteps, smm.ExecutedSteps)
	assert.Equal(t, 2, bem.GetFunctionCounter(getPendingBatch))
	assert.Equal(t, 2, bem.GetFunctionCounter(hasPendingBatch))
	assert.Equal(t, 3, bem.GetFunctionCounter(isLeader))
	assert.Equal(t, 1, bem.GetFunctionCounter(proposeTransferOnDestination))
	assert.Equal(t, 1, bem.GetFunctionCounter(printDebugInfo))
	assert.Equal(t, 1, bem.GetFunctionCounter(setStatusRejectedOnAllTransactions))
	assert.Equal(t, 3, bem.GetFunctionCounter(waitStepToFinish))
	assert.Equal(t, 0, bem.GetFunctionCounter(wasProposeTransferExecutedOnDestination))
	assert.Equal(t, 0, bem.GetFunctionCounter(signProposeTransferOnDestination))
	assert.Equal(t, 0, bem.GetFunctionCounter(isQuorumReachedForProposeTransfer))
	assert.Equal(t, 0, bem.GetFunctionCounter(executeTransferOnDestination))
	assert.Equal(t, 0, bem.GetFunctionCounter(wasTransferExecutedOnDestination))
	assert.Equal(t, 1, bem.GetFunctionCounter(cleanStoredSignatures))
	assert.Equal(t, 1, bem.GetFunctionCounter(updateTransactionsStatusesIfNeeded))
	assert.Equal(t, 1, bem.GetFunctionCounter(proposeSetStatusOnSource))
	assert.Equal(t, 1, bem.GetFunctionCounter(wasProposeSetStatusExecutedOnSource))
	assert.Equal(t, 1, bem.GetFunctionCounter(signProposeSetStatusOnSource))
	assert.Equal(t, 1, bem.GetFunctionCounter(isQuorumReachedForProposeSetStatus))
	assert.Equal(t, 1, bem.GetFunctionCounter(executeSetStatusOnSource))
	assert.Equal(t, 1, bem.GetFunctionCounter(wasSetStatusExecutedOnSource))
}

func TestFlowAsLeaderWasNotProposedTransferWithStubChecking(t *testing.T) {
	t.Parallel()

	bem := stateMachine.NewBridgeExecutorMock()
	setAllDecisionHandlersToTrue(bem)
	bem.WasProposeTransferExecutedOnDestinationCalled = falseHandlerWithContext

	steps, err := CreateSteps(bem)
	require.Nil(t, err)
	smm := stateMachine.NewStateMachineMock(steps, ethToElrond.GettingPending)
	err = smm.Initialize()
	require.Nil(t, err)

	numSteps := 6
	for i := 0; i < numSteps; i++ {
		err = smm.ExecuteOneStep()
		require.Nil(t, err)
	}

	expectedSteps := []core.StepIdentifier{
		ethToElrond.GettingPending,
		ethToElrond.ProposingTransfer,
		ethToElrond.ProposingTransfer,
		ethToElrond.ProposingTransfer,
		ethToElrond.ProposingTransfer,
		ethToElrond.ProposingTransfer,
	}

	assert.Equal(t, expectedSteps, smm.ExecutedSteps)
	assert.Equal(t, 1, bem.GetFunctionCounter(getPendingBatch))
	assert.Equal(t, 1, bem.GetFunctionCounter(hasPendingBatch))
	assert.Equal(t, 5, bem.GetFunctionCounter(isLeader))
	assert.Equal(t, 5, bem.GetFunctionCounter(proposeTransferOnDestination))
	assert.Equal(t, 5, bem.GetFunctionCounter(waitStepToFinish))
	assert.Equal(t, 5, bem.GetFunctionCounter(wasProposeTransferExecutedOnDestination))
}
