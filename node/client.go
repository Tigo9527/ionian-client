package node

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	providers "github.com/openweb3/go-rpc-provider/provider_wrapper"
	"github.com/sirupsen/logrus"
)

type Client struct {
	url string
	*providers.MiddlewarableProvider
}

func MustNewClient(url string, option ...providers.Option) *Client {
	client, err := NewClient(url, option...)
	if err != nil {
		logrus.WithError(err).WithField("url", url).Fatal("Failed to connect to storage node")
	}

	return client
}

func NewClient(url string, option ...providers.Option) (*Client, error) {
	var opt providers.Option
	if len(option) > 0 {
		opt = option[0]
	}

	provider, err := providers.NewProviderWithOption(url, opt)
	if err != nil {
		return nil, err
	}

	return &Client{
		url:                   url,
		MiddlewarableProvider: provider,
	}, nil
}

func MustNewClients(urls []string, option ...providers.Option) []*Client {
	var clients []*Client

	for _, url := range urls {
		client := MustNewClient(url, option...)
		clients = append(clients, client)
	}

	return clients
}

func (c *Client) URL() string {
	return c.url
}

func (c *Client) KV() *KvClient {
	return newKvClient(c.MiddlewarableProvider)
}

// Ionian RPCs

func (c *Client) GetStatus() (status Status, err error) {
	err = c.MiddlewarableProvider.CallContext(context.Background(), &status, "ionian_getStatus")
	return
}

func (c *Client) GetFileInfo(root common.Hash) (file *FileInfo, err error) {
	err = c.MiddlewarableProvider.CallContext(context.Background(), &file, "ionian_getFileInfo", root)
	return
}

func (c *Client) UploadSegment(segment SegmentWithProof) (ret int, err error) {
	err = c.MiddlewarableProvider.CallContext(context.Background(), &ret, "ionian_uploadSegment", segment)
	return
}

func (c *Client) DownloadSegment(root common.Hash, startIndex, endIndex uint32) (data []byte, err error) {
	err = c.MiddlewarableProvider.CallContext(context.Background(), &data, "ionian_downloadSegment", root, startIndex, endIndex)
	return
}

func (c *Client) DownloadSegmentWithProof(root common.Hash, index uint32) (segment *SegmentWithProof, err error) {
	err = c.MiddlewarableProvider.CallContext(context.Background(), &segment, "ionian_downloadSegmentWithProof", root, index)
	return
}

// Admin RPCs

func (c *Client) Shutdown() (ret int, err error) {
	err = c.MiddlewarableProvider.CallContext(context.Background(), &ret, "admin_shutdown")
	return
}

func (c *Client) StartSyncFile(txSeq uint64) (ret int, err error) {
	err = c.MiddlewarableProvider.CallContext(context.Background(), &ret, "admin_startSyncFile", txSeq)
	return
}

func (c *Client) GetSyncStatus(txSeq uint64) (status string, err error) {
	err = c.MiddlewarableProvider.CallContext(context.Background(), &status, "admin_getSyncStatus", txSeq)
	return
}
