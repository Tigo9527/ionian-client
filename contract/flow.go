package contract

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/openweb3/web3go"
)

type FlowExt struct {
	*contract
	*Flow
}

func NewFlowExt(flowAddress common.Address, clientWithSigner *web3go.Client) (*FlowExt, error) {
	backend, signer := clientWithSigner.ToClientForContract()

	contract, err := newContract(clientWithSigner, signer)
	if err != nil {
		return nil, err
	}

	flow, err := NewFlow(flowAddress, backend)
	if err != nil {
		return nil, err
	}

	return &FlowExt{contract, flow}, nil
}

func (flow *FlowExt) SubmitExt(submission IonianSubmission) (common.Hash, error) {
	opts, err := flow.CreateTransactOpts()
	if err != nil {
		return common.Hash{}, err
	}

	tx, err := flow.Submit(opts, submission)
	if err != nil {
		return common.Hash{}, err
	}

	return tx.Hash(), nil
}

func (submission IonianSubmission) String() string {
	var heights []uint64
	for _, v := range submission.Nodes {
		heights = append(heights, v.Height.Uint64())
	}

	return fmt.Sprintf("{ Size: %v, Heights: %v }", submission.Length, heights)
}
