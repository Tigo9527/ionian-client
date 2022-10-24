package kv

import (
	"math"
	"os"

	"github.com/Ionian-Web3-Storage/ionian-client/contract"
	"github.com/Ionian-Web3-Storage/ionian-client/file"
	"github.com/Ionian-Web3-Storage/ionian-client/node"
	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
)

// Client is used for users to communicate with server for kv operations.
type Client struct {
	node *node.Client
	flow *contract.Flow
}

// NewClient creates a new client for kv operations.
//
// Generally, you could refer to the `upload` function in `cmd/upload.go` file
// for how to create storage node client and flow contract client.
func NewClient(node *node.Client, flow *contract.Flow) *Client {
	return &Client{
		node: node,
		flow: flow,
	}
}

// Get returns paginated value for the specified stream key and offset.
func (c *Client) Get(streamId, key common.Hash, startIndex, length uint64, version ...uint64) (val *node.Value, err error) {
	return c.node.KV().GetValue(streamId, key, startIndex, length, version...)
}

func (c *Client) GetTransactionResult(txSeq uint64) (result string, err error) {
	return c.node.KV().GetTransactionResult(txSeq)
}

func (c *Client) GetHoldingStreamIds() (streamIds []common.Hash, err error) {
	return c.node.KV().GetHoldingStreamIds()
}

func (c *Client) HasWritePermission(account common.Address, streamId, key common.Hash, version ...uint64) (hasPermission bool, err error) {
	return c.node.KV().HasWritePermission(account, streamId, key, version...)
}

func (c *Client) IsAdmin(account common.Address, streamId common.Hash, version ...uint64) (isAdmin bool, err error) {
	return c.node.KV().IsAdmin(account, streamId, version...)
}

func (c *Client) IsSpecialKey(streamId, key common.Hash, version ...uint64) (isSpecialKey bool, err error) {
	return c.node.KV().IsSpecialKey(streamId, key, version...)
}

func (c *Client) IsWriterOfKey(account common.Address, streamId, key common.Hash, version ...uint64) (isWriter bool, err error) {
	return c.node.KV().IsWriterOfKey(account, streamId, key, version...)
}

func (c *Client) IsWriterOfStream(account common.Address, streamId common.Hash, version ...uint64) (isWriter bool, err error) {
	return c.node.KV().IsWriterOfStream(account, streamId, version...)
}

// Batcher returns a Batcher instance for kv operations in batch.
func (c *Client) Batcher() *Batcher {
	return newBatcher(math.MaxUint64, c)
}

type Batcher struct {
	*StreamDataBuilder
	client *Client
}

func newBatcher(version uint64, client *Client) *Batcher {
	return &Batcher{
		StreamDataBuilder: NewStreamDataBuilder(version),
		client:            client,
	}
}

// Exec submit the kv operations to Ionian network in batch.
//
// Note, this is a time consuming operation, e.g. several seconds or even longer.
// When it comes to a time sentitive context, it should be executed in a separate go-routine.
func (b *Batcher) Exec() error {
	// build stream data
	data, err := b.Build()
	if err != nil {
		return errors.WithMessage(err, "Failed to build stream data")
	}

	// prepare tmp file to upload
	tmpFilename, err := b.writeTempFile(data)
	if err != nil {
		return errors.WithMessage(err, "Failed to write stream data to temp file")
	}

	// upload file
	uploader := file.NewUploader(b.client.flow, b.client.node)
	if err = uploader.Upload(tmpFilename, b.BuildTags()); err != nil {
		return errors.WithMessagef(err, "Failed to upload file %v", tmpFilename)
	}

	// delete tmp file if completed
	return os.Remove(tmpFilename)
}

// writeTempFile encodes the specified stream data and write to a temp file.
//
// Note, the temp file should be removed via the returned temp file name.
func (b *Batcher) writeTempFile(data *StreamData) (string, error) {
	file, err := os.CreateTemp("", "ionian-kv-*")
	if err != nil {
		return "", errors.WithMessage(err, "Failed to create temp file")
	}
	defer file.Close()

	if _, err = file.Write(data.Encode()); err != nil {
		return "", errors.WithMessagef(err, "Failed to write data to %v", file.Name())
	}

	return file.Name(), nil
}
