package contracts

import (
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/contracts/blocksigner/contract"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"
	"math/big"
)

const (
	HexSignMethod = "2fb1b25f"
)

// Get ethClient over IPC of current node.
func GetEthClient(ctx *node.ServiceContext) (*ethclient.Client, error) {
	conf := ctx.GetConfig()
	client, err := ethclient.Dial(conf.IPCEndpoint())
	if err != nil {
		log.Error("Fail to connect RPC", "error", err)
		return nil, err
	}

	return client, nil
}

// Send tx sign for block number to smart contract blockSigner.
func CreateTransactionSign(chainConfig *params.ChainConfig, pool *core.TxPool, manager *accounts.Manager, block *types.Block) {
	// Find active account.
	account := accounts.Account{}
	var wallet accounts.Wallet
	if wallets := manager.Wallets(); len(wallets) > 0 {
		wallet = wallets[0]
		if accts := wallets[0].Accounts(); len(accts) > 0 {
			account = accts[0]
		}
	}

	// Create and send tx to smart contract for sign validate block.
	blockHex := common.LeftPadBytes(block.Number().Bytes(), 32)
	data := common.Hex2Bytes(HexSignMethod)
	inputData := append(data, blockHex...)
	nonce := pool.State().GetNonce(account.Address)
	tx := types.NewTransaction(nonce, common.HexToAddress(common.BlockSigners), big.NewInt(0), 100000, big.NewInt(0), inputData)
	txSigned, err := wallet.SignTx(account, tx, chainConfig.ChainId)
	if err != nil {
		log.Error("Fail to create tx sign", "error", err)
		return
	}

	// Add tx signed to local tx pool.
	pool.AddLocal(txSigned)
}

// Get signers signed for blockNumber from blockSigner contract.
func GetSignersFromContract(ctx *node.ServiceContext, blockNumber uint64) ([]common.Address, error) {
	client, err := GetEthClient(ctx)
	if err != nil {
		log.Error("Fail to connect IPC from blockSigner", "error", err)
		return nil, err
	}
	addr := common.HexToAddress(common.BlockSigners)
	blockSigner, err := contract.NewBlockSigner(addr, client)
	if err != nil {
		log.Error("Fail get block signers", "error", err)
		return nil, err
	}
	opts := new(bind.CallOpts)
	addrs, err := blockSigner.GetSigners(opts, new(big.Int).SetUint64(blockNumber))

	return addrs, nil
}