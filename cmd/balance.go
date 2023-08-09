/*
 */
package cmd

import (
	"fmt"
	"github.com/Ionian-Web3-Storage/ionian-client/contract"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	ethCommon "github.com/ethereum/go-ethereum/common"
	"github.com/openweb3/web3go"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"math/big"
	"time"
)

// balanceCmd represents the balance command
var balanceCmd = &cobra.Command{
	Use:   "balance",
	Short: "Check account balance, mint and approve.",
	Long:  `Check balance, native token (GAS) and storage token`,
	Run: func(cmd *cobra.Command, args []string) {
		client, contractAddr, flowExt, err := SetupLayer1(uploadArgs.url, uploadArgs.contract, uploadArgs.key)
		if err != nil {
			logrus.WithError(err).Fatal("Failed to setup layer1")
			return
		}
		defer client.Close()

		signer, _ := contract.DefaultSigner(client)
		account := signer.Address()
		balance, _ := client.Eth.Balance(account, nil)
		logrus.WithFields(logrus.Fields{
			"account": account, "balance": contract.ToDecimal(balance, 18),
		}).Info("account info")

		tokenBalance, err := getTokenInfo(&flowExt.FlowCaller, contractAddr, client)
		if err != nil {
			logrus.WithError(err).Error("Failed to get token info")
			return
		}
		tokenBalance.logTokenInfo()
	},
}

type TokenBalance struct {
	client          *web3go.Client
	account         ethCommon.Address
	tokenAddr       ethCommon.Address
	flowAddr        ethCommon.Address
	erc20caller     *contract.ERC20Caller
	balance         *big.Int
	fmtBalance      decimal.Decimal
	allowance       *big.Int
	fmtAllowance    decimal.Decimal
	erc20transactor *contract.ERC20Transactor
	decimals        int
}

func (tokenInfo *TokenBalance) checkAllowance(minBalance *big.Int) error {
	_, signFn := tokenInfo.client.ToClientForContract()
	fmtBalance := tokenInfo.fmtBalance
	token := tokenInfo.tokenAddr
	balance := tokenInfo.balance
	signer, _ := contract.DefaultSigner(tokenInfo.client)

	logrus.Debug("erc20 token balance ", fmtBalance.String())
	if balance.Cmp(minBalance) < 0 || balance.Cmp(big.NewInt(0)) == 0 {
		logrus.WithFields(logrus.Fields{
			"token": token.Hex(), "balance": balance, "minBalance": minBalance,
			"signer": signer.Address().Hex(),
		}).Warn("do not have enough balance")
		return fmt.Errorf("do not have enough balance")
	}

	if tokenInfo.allowance.Cmp(minBalance) < 0 {
		logrus.WithFields(logrus.Fields{
			"token": token.Hex(), "allowance": tokenInfo.allowance,
			"signer": signer.Address().Hex(), "minBalance": minBalance,
		}).Warn("do not have enough allowance, do approving now...")

		erc20 := tokenInfo.erc20transactor

		tx, err := erc20.Approve(&bind.TransactOpts{
			Signer: signFn, From: signer.Address(),
		}, tokenInfo.flowAddr, minBalance)
		if err != nil {
			return errors.WithMessage(err, "Approve fail")
		}
		logrus.Info("approve tx, waiting... ", tx.Hash().Hex())
		rcpt, err := contract.WaitForReceipt(tokenInfo.client, tx.Hash(), true, time.Second)
		if err != nil {
			return errors.WithMessage(err, "Approve tx fail")
		}
		logrus.Info("approved. receipt status ", *rcpt.Status)
	}
	return nil
}

func (tokenInfo *TokenBalance) logTokenInfo() {
	logrus.WithFields(logrus.Fields{
		"fmtBalance":   tokenInfo.fmtBalance.String(),
		"fmtAllowance": tokenInfo.fmtAllowance.String(),
		"token":        tokenInfo.tokenAddr.Hex(),
		"spender":      tokenInfo.flowAddr.Hex(),
	}).Info("token info")
}

func getTokenInfo(flowCaller *contract.FlowCaller, flowAddr *ethCommon.Address, client *web3go.Client) (*TokenBalance, error) {
	callOpts := &bind.CallOpts{}
	token, err := flowCaller.Token(callOpts)
	if err != nil {
		return nil, errors.WithMessage(err, "get token of flow fail")
	}
	signer, _ := contract.DefaultSigner(client)
	backend, _ := client.ToClientForContract()
	erc20caller, err := contract.NewERC20Caller(token, backend)
	if err != nil {
		return nil, errors.WithMessage(err, "NewERC20Caller fail")
	}
	balance, err := erc20caller.BalanceOf(callOpts, signer.Address())
	if err != nil {
		return nil, errors.WithMessage(err, "BalanceOf fail")
	}
	decimals, _ := erc20caller.Decimals(callOpts)
	fmtBalance := contract.ToDecimal(balance, int(decimals))

	allowance, err := erc20caller.Allowance(callOpts, signer.Address(), *flowAddr)
	if err != nil {
		return nil, errors.WithMessage(err, "Allowance fail")
	}
	fmtAllowance := contract.ToDecimal(allowance, int(decimals))

	erc20transactor, err := contract.NewERC20Transactor(token, backend)
	if err != nil {
		return nil, errors.WithMessage(err, "NewERC20Transactor fail")
	}

	return &TokenBalance{
		client: client, account: signer.Address(),
		tokenAddr: token, erc20caller: erc20caller,
		balance: balance, fmtBalance: fmtBalance,
		allowance: allowance, fmtAllowance: fmtAllowance, erc20transactor: erc20transactor,
		decimals: int(decimals),
		flowAddr: *flowAddr,
	}, err
}

func init() {
	rootCmd.AddCommand(balanceCmd)
	Layer1args(balanceCmd)
}
