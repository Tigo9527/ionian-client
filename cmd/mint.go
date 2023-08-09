package cmd

import (
	"github.com/Ionian-Web3-Storage/ionian-client/contract"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/sirupsen/logrus"
	"math/big"
	"time"

	"github.com/spf13/cobra"
)

// mintCmd represents the mint command
var mintCmd = &cobra.Command{
	Use:   "mint",
	Short: "Mint faucet token",
	Long:  `Call mint(address, amount) on the token contract used by ionian.`,
	Run: func(cmd *cobra.Command, args []string) {
		client, contractAddr, flowExt, err := SetupLayer1(uploadArgs.url, uploadArgs.contract, uploadArgs.key)
		if err != nil {
			logrus.WithError(err).Fatal("Failed to setup layer1")
			return
		}
		defer client.Close()

		tokenInfo, err := getTokenInfo(&flowExt.FlowCaller, contractAddr, client)
		if err != nil {
			logrus.WithError(err).Fatal("Failed to get token info")
			return
		}
		tokenInfo.logTokenInfo()
		balance, _ := client.Eth.Balance(tokenInfo.account, nil)
		if balance.Cmp(big.NewInt(0)) == 0 {
			logrus.Fatal("Native token is zero, account ", tokenInfo.account.Hex())
			return
		}

		var amount, e = big.NewInt(10), big.NewInt(int64(tokenInfo.decimals + 2))
		amount.Exp(amount, e, nil)
		_, signFn := client.ToClientForContract()
		tx, err := tokenInfo.erc20transactor.Mint(&bind.TransactOpts{
			Signer: signFn, From: tokenInfo.account,
		}, tokenInfo.account, amount)
		if err != nil {
			logrus.WithError(err).Fatal("Failed to mint")
			return
		}
		logrus.Info("waiting for mint tx ", tx.Hash().Hex())
		_, err = contract.WaitForReceipt(client, tx.Hash(), true, time.Second)
		if err != nil {
			logrus.WithError(err).Fatal("Failed to WaitForReceipt")
			return
		}
		logrus.Info("Minted. tx ", tx.Hash().Hex())
		_ = tokenInfo.checkAllowance(amount)
		tokenInfo, _ = getTokenInfo(&flowExt.FlowCaller, contractAddr, client)
		tokenInfo.logTokenInfo()
	},
}

func init() {
	balanceCmd.AddCommand(mintCmd)
	Layer1args(mintCmd)
}
