package contract

import (
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/openweb3/web3go"
	"github.com/openweb3/web3go/types"
)

var CustomGasPrice uint64
var CustomGasLimit uint64

type contract struct {
	client  *web3go.Client
	account common.Address
	signer  bind.SignerFn
}

func newContract(clientWithSigner *web3go.Client, signerFn bind.SignerFn) (*contract, error) {
	signer, err := DefaultSigner(clientWithSigner)
	if err != nil {
		return nil, err
	}

	return &contract{
		client:  clientWithSigner,
		account: signer.Address(),
		signer:  signerFn,
	}, nil
}

func (c *contract) CreateTransactOpts() (*bind.TransactOpts, error) {
	var gasPrice *big.Int
	if CustomGasPrice > 0 {
		gasPrice = new(big.Int).SetUint64(CustomGasPrice)
	}

	return &bind.TransactOpts{
		From:     c.account,
		GasPrice: gasPrice,
		GasLimit: CustomGasLimit,
		Signer:   c.signer,
	}, nil
}

func (c *contract) WaitForReceipt(txHash common.Hash, successRequired bool, pollInterval ...time.Duration) (*types.Receipt, error) {
	return WaitForReceipt(c.client, txHash, successRequired, pollInterval...)
}
