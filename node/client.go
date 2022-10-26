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

	ionian *IonianClient
	admin  *AdminClient
	kv     *KvClient
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

		ionian: &IonianClient{provider},
		admin:  &AdminClient{provider},
		kv:     &KvClient{provider},
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

func (c *Client) Ionian() *IonianClient {
	return c.ionian
}

func (c *Client) Admin() *AdminClient {
	return c.admin
}

func (c *Client) KV() *KvClient {
	return c.kv
}

// Ionian RPCs
type IonianClient struct {
	provider *providers.MiddlewarableProvider
}

func (c *IonianClient) GetStatus() (status Status, err error) {
	err = c.provider.CallContext(context.Background(), &status, "ionian_getStatus")
	return
}

func (c *IonianClient) GetFileInfo(root common.Hash) (file *FileInfo, err error) {
	err = c.provider.CallContext(context.Background(), &file, "ionian_getFileInfo", root)
	return
}

func (c *IonianClient) GetFileInfoByTxSeq(txSeq uint64) (file *FileInfo, err error) {
	err = c.provider.CallContext(context.Background(), &file, "ionian_getFileInfoByTxSeq", txSeq)
	return
}

func (c *IonianClient) UploadSegment(segment SegmentWithProof) (ret int, err error) {
	err = c.provider.CallContext(context.Background(), &ret, "ionian_uploadSegment", segment)
	return
}

func (c *IonianClient) DownloadSegment(root common.Hash, startIndex, endIndex uint64) (data []byte, err error) {
	err = c.provider.CallContext(context.Background(), &data, "ionian_downloadSegment", root, startIndex, endIndex)
	return
}

func (c *IonianClient) DownloadSegmentWithProof(root common.Hash, index uint64) (segment *SegmentWithProof, err error) {
	err = c.provider.CallContext(context.Background(), &segment, "ionian_downloadSegmentWithProof", root, index)
	return
}

// Admin RPCs
type AdminClient struct {
	provider *providers.MiddlewarableProvider
}

func (c *AdminClient) Shutdown() (ret int, err error) {
	err = c.provider.CallContext(context.Background(), &ret, "admin_shutdown")
	return
}

func (c *AdminClient) StartSyncFile(txSeq uint64) (ret int, err error) {
	err = c.provider.CallContext(context.Background(), &ret, "admin_startSyncFile", txSeq)
	return
}

func (c *AdminClient) GetSyncStatus(txSeq uint64) (status string, err error) {
	err = c.provider.CallContext(context.Background(), &status, "admin_getSyncStatus", txSeq)
	return
}
