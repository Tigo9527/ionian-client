package node

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	providers "github.com/openweb3/go-rpc-provider/provider_wrapper"
)

type KvClient struct {
	provider *providers.MiddlewarableProvider
}

func newKvClient(provider *providers.MiddlewarableProvider) *KvClient {
	return &KvClient{provider}
}

func (c *KvClient) GetValue(streamId, key common.Hash, startIndex, length uint64, version ...uint64) (val *Value, err error) {
	args := []interface{}{streamId, key, startIndex, length}
	if len(version) > 0 {
		args = append(args, version[0])
	}
	err = c.provider.CallContext(context.Background(), &val, "kv_getValue", args...)
	return
}
