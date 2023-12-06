package batchProcessor

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/multiversx/mx-bridge-eth-go/clients"
	"math/big"
)

type ArgListsBatch struct {
	Tokens              []common.Address
	Recipients          []common.Address
	ConvertedTokenBytes [][]byte
	Amounts             []*big.Int
	Nonces              []*big.Int
}

func ExtractList(batch *clients.TransferBatch) (*ArgListsBatch, error) {
	arg := ArgListsBatch{}

	for _, dt := range batch.Deposits {
		recipient := common.BytesToAddress(dt.ToBytes)
		arg.Recipients = append(arg.Recipients, recipient)

		token := common.BytesToAddress(dt.ConvertedTokenBytes)
		arg.Tokens = append(arg.Tokens, token)

		amount := big.NewInt(0).Set(dt.Amount)
		arg.Amounts = append(arg.Amounts, amount)

		nonce := big.NewInt(0).SetUint64(dt.Nonce)
		arg.Nonces = append(arg.Nonces, nonce)

		arg.ConvertedTokenBytes = append(arg.ConvertedTokenBytes, dt.ConvertedTokenBytes)
	}

	return &arg, nil
}
