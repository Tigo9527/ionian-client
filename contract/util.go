package contract

import (
	"encoding/json"
	"io/ioutil"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/openweb3/web3go"
	"github.com/openweb3/web3go/interfaces"
	"github.com/openweb3/web3go/types"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"
)

func WaitForReceipt(client *web3go.Client, txHash common.Hash, successRequired bool, pollInterval ...time.Duration) (receipt *types.Receipt, err error) {
	interval := time.Second
	if len(pollInterval) > 0 && pollInterval[0] > 0 {
		interval = pollInterval[0]
	}

	for receipt == nil {
		time.Sleep(interval)

		if receipt, err = client.Eth.TransactionReceipt(txHash); err != nil {
			return nil, err
		}
	}

	if receipt.Status == nil {
		return nil, errors.New("Status not found in receipt")
	}

	switch *receipt.Status {
	case gethTypes.ReceiptStatusSuccessful:
		return receipt, nil
	case gethTypes.ReceiptStatusFailed:
		if !successRequired {
			return receipt, nil
		}

		if receipt.TxExecErrorMsg == nil {
			return nil, errors.New("Transaction execution failed")
		}

		return nil, errors.Errorf("Transaction execution failed, %v", *receipt.TxExecErrorMsg)
	default:
		return nil, errors.Errorf("Unknown receipt status %v", *receipt.Status)
	}
}

func ToDecimal(value *big.Int, decimals int) decimal.Decimal {
	mul := decimal.NewFromFloat(float64(10)).Pow(decimal.NewFromFloat(float64(decimals)))
	num, _ := decimal.NewFromString(value.String())
	result := num.Div(mul)

	return result
}

func DefaultSigner(clientWithSigner *web3go.Client) (interfaces.Signer, error) {
	sm, err := clientWithSigner.GetSignerManager()
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to get signer manager from client")
	}

	if sm == nil {
		return nil, errors.New("Signer not specified")
	}

	signers := sm.List()
	if len(signers) == 0 {
		return nil, errors.WithMessage(err, "Account not configured in signer manager")
	}

	return signers[0], nil
}

func Deploy(clientWithSigner *web3go.Client, dataOrFile string) (common.Address, error) {
	signer, err := DefaultSigner(clientWithSigner)
	if err != nil {
		return common.Address{}, errors.WithMessage(err, "Failed to detect account")
	}
	from := signer.Address()

	bytecode, err := parseBytecode(dataOrFile)
	if err != nil {
		return common.Address{}, errors.WithMessage(err, "Failed to parse bytecode")
	}

	var gasPrice *hexutil.Big
	if CustomGasPrice > 0 {
		gasPrice = (*hexutil.Big)(new(big.Int).SetUint64(CustomGasPrice))
	}

	var gasLimit *hexutil.Uint64
	if CustomGasLimit > 0 {
		gasLimit = (*hexutil.Uint64)(&CustomGasLimit)
	}

	txHash, err := clientWithSigner.Eth.SendTransactionByArgs(types.TransactionArgs{
		From:     &from,
		Data:     &bytecode,
		GasPrice: gasPrice,
		Gas:      gasLimit,
	})
	if err != nil {
		return common.Address{}, errors.WithMessage(err, "Failed to send transaction")
	}

	logrus.WithField("hash", txHash).Info("Transaction sent to blockchain")

	receipt, err := WaitForReceipt(clientWithSigner, txHash, true)
	if err != nil {
		return common.Address{}, errors.WithMessage(err, "Failed to wait for receipt")
	}

	return *receipt.ContractAddress, nil
}

func parseBytecode(dataOrFile string) (hexutil.Bytes, error) {
	if strings.HasPrefix(dataOrFile, "0x") {
		return hexutil.Decode(dataOrFile)
	}

	content, err := ioutil.ReadFile(dataOrFile)
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to read file")
	}

	var data map[string]interface{}
	if err = json.Unmarshal(content, &data); err != nil {
		return nil, errors.WithMessage(err, "Failed to unmarshal JSON")
	}

	bytecode, ok := data["bytecode"]
	if !ok {
		return nil, errors.New("bytecode field not found in JSON file")
	}

	if bytecodeStr, ok := bytecode.(string); ok {
		return hexutil.Decode(bytecodeStr)
	}

	bytecodeObj, ok := bytecode.(map[string]interface{})
	if !ok {
		return nil, errors.New("invalid type for bytecode field")
	}

	bytecode, ok = bytecodeObj["object"]
	if !ok {
		return nil, errors.New("bytecode.object field not found in JSON file")
	}

	if bytecodeStr, ok := bytecode.(string); ok {
		return hexutil.Decode(bytecodeStr)
	}

	return nil, errors.New("invalid type for bytecode field")
}

func ConvertToGethLog(log *types.Log) *gethTypes.Log {
	if log == nil {
		return nil
	}

	return &gethTypes.Log{
		Address:     log.Address,
		Topics:      log.Topics,
		Data:        log.Data,
		BlockNumber: log.BlockNumber,
		TxHash:      log.TxHash,
		TxIndex:     log.TxIndex,
		BlockHash:   log.BlockHash,
		Index:       log.Index,
		Removed:     log.Removed,
	}
}
