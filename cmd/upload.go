package cmd

import (
	"github.com/Ionian-Web3-Storage/ionian-client/common"
	"github.com/Ionian-Web3-Storage/ionian-client/contract"
	"github.com/Ionian-Web3-Storage/ionian-client/file"
	"github.com/Ionian-Web3-Storage/ionian-client/node"
	ethCommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/openweb3/web3go"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	uploadArgs struct {
		file string
		tags string

		url      string
		contract string
		key      string

		node string

		force bool
	}

	uploadCmd = &cobra.Command{
		Use:   "upload",
		Short: "Upload file to Ionian network",
		Run:   upload,
	}
)

func Layer1args(cmd *cobra.Command) {
	cmd.Flags().StringVar(&uploadArgs.url, "url", "", "Fullnode URL to interact with Ionian smart contract")
	cmd.MarkFlagRequired("url")
	cmd.Flags().StringVar(&uploadArgs.contract, "contract", "", "Ionian smart contract to interact with")
	cmd.MarkFlagRequired("contract")
	cmd.Flags().StringVar(&uploadArgs.key, "key", "", "Private key to interact with smart contract")
	cmd.MarkFlagRequired("key")
}

func init() {
	uploadCmd.Flags().StringVar(&uploadArgs.file, "file", "", "File name to upload")
	uploadCmd.MarkFlagRequired("file")
	uploadCmd.Flags().StringVar(&uploadArgs.tags, "tags", "0x", "Tags of the file")

	Layer1args(uploadCmd)

	uploadCmd.Flags().StringVar(&uploadArgs.node, "node", "", "Ionian storage node URL")
	uploadCmd.MarkFlagRequired("node")

	uploadCmd.Flags().BoolVar(&uploadArgs.force, "force", false, "Force to upload file even already exists")

	rootCmd.AddCommand(uploadCmd)
}

func checkToken(flowCaller *contract.FlowCaller, flowAddr *ethCommon.Address, client *web3go.Client) error {
	tokenInfo, err := getTokenInfo(flowCaller, flowAddr, client)
	if err != nil {
		return err
	}
	tokenInfo.logTokenInfo()
	return nil
}

func SetupLayer1(url string, contractStr string, key string) (*web3go.Client, *ethCommon.Address, *contract.FlowExt, error) {
	client := common.MustNewWeb3(url, key)
	contractAddr := ethCommon.HexToAddress(contractStr)
	flow, err := contract.NewFlowExt(contractAddr, client)
	if err != nil {
		return nil, nil, nil, errors.WithMessage(err, "Failed to create flow contract")
	}
	return client, &contractAddr, flow, nil
}

func upload(*cobra.Command, []string) {
	client, contractAddr, flow, err := SetupLayer1(uploadArgs.url, uploadArgs.contract, uploadArgs.key)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to setup layer1")
		return
	}
	defer client.Close()

	err = checkToken(&flow.FlowCaller, contractAddr, client)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to check token balance")
		return
	}

	node := node.MustNewClient(uploadArgs.node)
	defer node.Close()

	uploader := file.NewUploader(flow, node)
	opt := file.UploadOption{
		Tags:  hexutil.MustDecode(uploadArgs.tags),
		Force: uploadArgs.force,
	}
	if err := uploader.Upload(uploadArgs.file, opt); err != nil {
		logrus.WithError(err).Fatal("Failed to upload file")
	}
}
