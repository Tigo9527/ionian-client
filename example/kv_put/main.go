package main

import (
	"fmt"

	"github.com/Ionian-Web3-Storage/ionian-client/common"
	"github.com/Ionian-Web3-Storage/ionian-client/contract"
	"github.com/Ionian-Web3-Storage/ionian-client/kv"
	"github.com/Ionian-Web3-Storage/ionian-client/node"
	ethCommon "github.com/ethereum/go-ethereum/common"
)

const IonianClientAddr = "http://127.0.0.1:5678"
const BlockchainClientAddr = ""
const PrivKey = ""
const FlowContractAddr = ""

func main() {
	ionianClient, err := node.NewClient(IonianClientAddr)
	if err != nil {
		fmt.Println(err)
		return
	}
	blockchainClient := common.MustNewWeb3(BlockchainClientAddr, PrivKey)
	defer blockchainClient.Close()
	contract.CustomGasLimit = 1000000
	ionian, err := contract.NewFlowExt(ethCommon.HexToAddress(FlowContractAddr), blockchainClient)
	if err != nil {
		fmt.Println(err)
		return
	}
	kvClient := kv.NewClient(ionianClient, ionian)
	batcher := kvClient.Batcher()
	batcher.Set(ethCommon.HexToHash("0x000000000000000000000000000000000000000000000000000000000000f2bd"),
		[]byte("TESTKEY0"),
		[]byte{69, 70, 71, 72, 73},
	)
	batcher.Set(ethCommon.HexToHash("0x000000000000000000000000000000000000000000000000000000000000f2bd"),
		[]byte("TESTKEY1"),
		[]byte{74, 75, 76, 77, 78},
	)
	err = batcher.Exec()
	if err != nil {
		fmt.Println(err)
		return
	}
}
