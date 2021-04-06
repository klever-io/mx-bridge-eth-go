package eth

import (
	"context"
	"github.com/ElrondNetwork/elrond-eth-bridge/safe"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"math/big"
	"reflect"
	"strings"
	"testing"
)

// verify Client implements interface
var (
	_ = safe.Safe(&Client{})
)

type testChainReader struct {
	blocks []types.Block
}

func (c *testChainReader) addBlockFromHex(hex string) error {
	blockEnc := common.FromHex(hex)
	var block types.Block
	err := rlp.DecodeBytes(blockEnc, &block)

	if err != nil {
		return err
	}

	c.blocks = append(c.blocks, block)
	return nil
}

func (c *testChainReader) BlockByHash(context.Context, common.Hash) (*types.Block, error) {
	return nil, nil
}
func (c *testChainReader) BlockByNumber(_ context.Context, number *big.Int) (*types.Block, error) {
	if number.Int64() >= int64(len(c.blocks)) {
		return nil, ethereum.NotFound
	}
	return &c.blocks[number.Int64()], nil
}
func (c *testChainReader) HeaderByHash(context.Context, common.Hash) (*types.Header, error) {
	return nil, nil
}
func (c *testChainReader) HeaderByNumber(context.Context, *big.Int) (*types.Header, error) {
	return nil, nil
}
func (c *testChainReader) TransactionCount(context.Context, common.Hash) (uint, error) {
	return 0, nil
}
func (c *testChainReader) TransactionInBlock(context.Context, common.Hash, uint) (*types.Transaction, error) {
	return nil, nil
}

func (c *testChainReader) SubscribeNewHead(context.Context, chan<- *types.Header) (ethereum.Subscription, error) {
	return nil, nil
}

type testBlockstorer struct {
	lastBlockIndexStored *big.Int
}

func (b *testBlockstorer) StoreBlockIndex(index *big.Int) error {
	b.lastBlockIndexStored = index
	return nil
}

func TestGetTransactions(t *testing.T) {
	chainReader := &testChainReader{}
	// amount 2
	err := chainReader.addBlockFromHex(`f902a8f901f6a01bbf9582f7751e9493e5cf64396c04ea6dd0738fbce2cd1aa4c270e4cfd4a149a01dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347940000000000000000000000000000000000000000a0ad6377e310e8a0cdc9277c5ad4c7649e244fc0accd5c0e99f26e9813a51781a2a0e9a78698cccbce3e0e2bfd94ae8c0603e4b29e39523ea1059e670ee2d4e8117ba01f05c40ba7a935d1492757fa666ea1eac4924aa4acd1540f523bad3d1e0d5d46b9010000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000000000000000000000000000000008000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000001000000000000000000000000000000800f836691b7825e78846060ba4980a00000000000000000000000000000000000000000000000000000000000000000880000000000000000f8acf8aa0e8504a817c8008301d858946224dde04296e2528ef5c5705db49bfcbf04372180b84447e7ef240000000000000000000000005abc5e20f56dc6ce962c458a3142fc289a757f4e000000000000000000000000000000000000000000000000000000000000000226a0d674cc92445a68a121b723c6ce8485799d278f9570872af95634e0735e8f07e7a022fcf034aec13be9c703313855e95aa9112321d71c77cd17428b9da20693cb4fc0`)
	assertDecoding(t, err)
	// no safe calls
	err = chainReader.addBlockFromHex(`f9026cf901f6a006f781b6f6c303ac967fe6bc17fd74f23294231313a4ef05a3636c3f916a6569a01dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347940000000000000000000000000000000000000000a07fb924431056423d73942818445ff178484efd4336df6a29f4cc11e29c953942a06d3490b8e60bc841029738cf0ae3ee58cbd6e7dd36bc2d1416123b91f780e75ba0056b23fbba480696b65fe5a59b8f2148a1299103c4f57df839233af2cf4ca2d2b90100000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000008011836691b782520884606160c680a00000000000000000000000000000000000000000000000000000000000000000880000000000000000f870f86e108504a817c8008252089449253d72bcc7b531f1b399983bc30e2ec5ec7ff4884563918244f4000080820a95a0070a260f4b40bf46ee455c60a4487ce55b67c677f602ee6c73dc2816796d1934a064b117fae533b962b90282ac625f0a7d34687e0b5b9ec6282e3bf528e1e36825c0`)
	assertDecoding(t, err)
	// amount 1
	err = chainReader.addBlockFromHex(`f902a8f901f6a0d2827949983edad89f9c2b26d1e7e27e350e7601f152281041c7884953e8fc13a01dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347940000000000000000000000000000000000000000a021a3dcc052faaceff9b4cea40cceb375aeda339cd183e125eca8e3623ef1f628a0d027e2ef0e9863d812c1f04522dd30d733eb6e307a439de2d969cab8c64a7727a08c452bfa95b85ff54bb7dbef02878a385a9079659205bf7e1cfeb42653cbcfa3b90100000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000008000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000040000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000010000000000000000000000000000008010836691b7825e788460615b7780a00000000000000000000000000000000000000000000000000000000000000000880000000000000000f8acf8aa0f8504a817c8008301d858946224dde04296e2528ef5c5705db49bfcbf04372180b84447e7ef240000000000000000000000005abc5e20f56dc6ce962c458a3142fc289a757f4e000000000000000000000000000000000000000000000000000000000000000126a07345f2fe01dd478568628ee9fa85a430175db7f343c5d098af95ef792e6aabf9a06854bb33b49dae83db6efd4ad623c51cc19b13aad9631ee2cc16b8248ade3cd4c0`)
	assertDecoding(t, err)

	mostRecentBlockNumber := func(ctx context.Context) (*big.Int, error) {
		return big.NewInt(int64(len(chainReader.blocks) - 1)), nil
	}
	safeAbi, err := abi.JSON(strings.NewReader(safeAbiDefinition))

	if err != nil {
		t.Fatal(err)
	}

	blockstorer := &testBlockstorer{}
	client := Client{
		chainReader:           chainReader,
		blockstorer:           blockstorer,
		safeAddress:           common.HexToAddress("0x6224Dde04296e2528eF5C5705Db49bfCbF043721"),
		safeAbi:               safeAbi,
		mostRecentBlockNumber: mostRecentBlockNumber,
	}
	channel := make(safe.SafeTxChan)
	go client.GetTransactions(context.Background(), big.NewInt(0), channel)

	t1 := <-channel
	t2 := <-channel

	wantT1 := &safe.DepositTransaction{
		Hash:         "0x3073d1b5aaa4c26b892cbe534f0a7d185535a5f46028d85425454298abf8d903",
		From:         "0x5246eb39712BA66357cc5c0d77Bd737e62FbC534",
		TokenAddress: "0x5abc5e20F56Dc6Ce962C458A3142FC289A757F4E",
		Amount:       big.NewInt(2),
	}

	wantT2 := &safe.DepositTransaction{
		Hash:         "0xe0142af981864d535634bf25b997304e231659f5156d52ce4a0a3f632e72138b",
		From:         "0x5246eb39712BA66357cc5c0d77Bd737e62FbC534",
		TokenAddress: "0x5abc5e20F56Dc6Ce962C458A3142FC289A757F4E",
		Amount:       big.NewInt(1),
	}

	var transactionTests = []struct {
		name   string
		got    *safe.DepositTransaction
		wanted *safe.DepositTransaction
	}{
		{"transaction with 2 tokens", t1, wantT1},
		{"transaction with 1 token", t2, wantT2},
	}

	for _, tt := range transactionTests {
		t.Run(tt.name, func(t *testing.T) {
			if !reflect.DeepEqual(t1, wantT1) {
				t.Errorf("Wanted %v, got %v", tt.wanted, tt.got)
			}
		})
	}

	if !reflect.DeepEqual(blockstorer.lastBlockIndexStored, big.NewInt(3)) {
		t.Errorf("Expected last stored block index to be %v, but was %v", 3, blockstorer.lastBlockIndexStored)
	}
}

func assertDecoding(t testing.TB, err error) {
	t.Helper()

	if err != nil {
		t.Fatal("Failed to decode block")
	}
}
