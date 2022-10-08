package file

import (
	"time"

	"github.com/Ionian-Web3-Storage/ionian-client/contract"
	"github.com/Ionian-Web3-Storage/ionian-client/file/merkle"
	"github.com/Ionian-Web3-Storage/ionian-client/node"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// maxDataSize is the maximum data size to upload on blockchain directly.
// const maxDataSize = int64(4 * 1024)

type Uploader struct {
	ionian *contract.Flow
	client *node.Client
}

func NewUploader(ionian *contract.Flow, client *node.Client) *Uploader {
	return &Uploader{
		ionian: ionian,
		client: client,
	}
}

func NewUploaderLight(client *node.Client) *Uploader {
	return &Uploader{
		client: client,
	}
}

func (uploader *Uploader) Upload(filename string, tags []byte) error {
	// Open file to upload
	file, err := Open(filename)
	if err != nil {
		return errors.WithMessage(err, "Failed to open file")
	}
	defer file.Close()

	logrus.WithFields(logrus.Fields{
		"name":     file.Name(),
		"size":     file.Size(),
		"chunks":   file.NumChunks(),
		"segments": file.NumSegments(),
	}).Info("File prepared to upload")

	// Calculate file merkle root.
	tree, err := file.MerkleTree()
	if err != nil {
		return errors.WithMessage(err, "Failed to create file merkle tree")
	}
	logrus.WithField("root", tree.Root()).Info("File merkle root calculated")

	info, err := uploader.client.GetFileInfo(tree.Root())
	if err != nil {
		return errors.WithMessage(err, "Failed to get file info from storage node")
	}

	logrus.WithField("info", info).Debug("Log entry retrieved from storage node")

	if uploader.ionian == nil && info == nil {
		return errors.New("log entry not available on storage node")
	}

	// Upload small data on blockchain directly.
	// if file.Size() <= maxDataSize {
	// 	if info != nil {
	// 		return errors.New("File already exists on Ionian network")
	// 	}

	// 	return uploader.uploadSmallData(filename)
	// }

	if info != nil && info.Finalized {
		return errors.New("File already exists on Ionian network")
	}

	if info == nil {
		// Append log on blockchain
		if err = uploader.submitLogEntry(file, tags, tree); err != nil {
			return errors.WithMessage(err, "Failed to submit log entry")
		}

		// Wait for storage node to retrieve log entry from blockchain
		if err = uploader.waitForLogEntry(tree.Root()); err != nil {
			return errors.WithMessage(err, "Failed to check if log entry available on storage node")
		}
	}

	// Upload file to storage node
	if err = uploader.uploadFile(file, tree); err != nil {
		return errors.WithMessage(err, "Failed to upload file")
	}

	// Wait for transaction finality
	if err = uploader.waitForFinality(tree.Root()); err != nil {
		return errors.WithMessage(err, "Failed to wait for transaction finality on storage node")
	}

	return nil
}

// func (uploader *Uploader) uploadSmallData(filename string) error {
// 	content, err := ioutil.ReadFile(filename)
// 	if err != nil {
// 		return errors.WithMessage(err, "Failed to read data from file")
// 	}

// 	hash, err := uploader.ionian.AppendLogWithData(content)
// 	if err != nil {
// 		return errors.WithMessage(err, "Failed to send transaction to append log with data")
// 	}

// 	logrus.WithField("hash", hash.Hex()).Info("Succeeded to send transaction to append log with data")

// 	return uploader.waitForSuccessfulExecution(hash)
// }

func (uploader *Uploader) waitForSuccessfulExecution(txHash common.Hash) error {
	logrus.WithField("tx", txHash).Info("Wait for transaction execution")

	receipt, err := uploader.ionian.WaitForReceipt(txHash)
	if err != nil {
		return errors.WithMessage(err, "Failed to wait for receipt")
	}

	if receipt.Status == nil {
		return errors.New("status not found in receipt")
	}

	switch *receipt.Status {
	case types.ReceiptStatusSuccessful:
		return nil
	case types.ReceiptStatusFailed:
		if receipt.TxExecErrorMsg == nil {
			return errors.New("Transaction execution failed")
		}

		return errors.Errorf("Transaction execution failed, %v", *receipt.TxExecErrorMsg)
	default:
		return errors.Errorf("Unknown receipt status %v", *receipt.Status)
	}
}

func (uploader *Uploader) submitLogEntry(file *File, tags []byte, tree *merkle.Tree) error {
	flow := NewFlow(file, tags)
	submission, err := flow.CreateSubmission()
	if err != nil {
		return errors.WithMessage(err, "Failed to create flow submission")
	}

	// Submit log entry to smart contract.
	hash, err := uploader.ionian.Submit(*submission)
	if err != nil {
		return errors.WithMessage(err, "Failed to send transaction to append log entry")
	}

	logrus.WithField("hash", hash.Hex()).Info("Succeeded to send transaction to append log entry")

	return uploader.waitForSuccessfulExecution(hash)
}

// Wait for log entry ready on storage node.
func (uploader *Uploader) waitForLogEntry(root common.Hash) error {
	logrus.WithField("root", root).Info("Wait for log entry on storage node")

	for {
		time.Sleep(time.Second)

		info, err := uploader.client.GetFileInfo(root)
		if err != nil {
			return errors.WithMessage(err, "Failed to get file info from storage node")
		}

		if info != nil {
			break
		}
	}

	return nil
}

// TODO error tolerance
func (uploader *Uploader) uploadFile(file *File, tree *merkle.Tree) error {
	logrus.Info("Begin to upload file")

	iter := file.Iterate(true)
	var segIndex int

	for {
		ok, err := iter.Next()
		if err != nil {
			return errors.WithMessage(err, "Failed to read segment")
		}

		if !ok {
			break
		}

		segment := iter.Current()
		proof := tree.ProofAt(segIndex)

		// Skip upload rear padding data
		numChunks := file.NumChunks()
		startIndex := segIndex * DefaultSegmentMaxChunks
		allDataUploaded := false
		if startIndex >= int(numChunks) {
			// file real data already uploaded
			break
		} else if startIndex+len(segment)/DefaultChunkSize >= int(numChunks) {
			// last segment has real data
			expectedLen := DefaultChunkSize * (int(numChunks) - startIndex)
			segment = segment[:expectedLen]
			allDataUploaded = true
		}

		segWithProof := node.SegmentWithProof{
			Root:     tree.Root(),
			Data:     segment,
			Index:    uint32(segIndex),
			Proof:    proof,
			FileSize: uint64(file.Size()),
		}

		if _, err = uploader.client.UploadSegment(segWithProof); err != nil {
			return errors.WithMessage(err, "Failed to upload segment")
		}

		if logrus.IsLevelEnabled(logrus.DebugLevel) {
			chunkIndex := segIndex * DefaultSegmentMaxChunks
			logrus.WithFields(logrus.Fields{
				"total":      file.NumSegments(),
				"index":      segIndex,
				"chunkStart": chunkIndex,
				"chunkEnd":   chunkIndex + len(segment)/DefaultChunkSize,
				"root":       segmentRoot(segment),
			}).Debug("Segment uploaded")
		}

		if allDataUploaded {
			break
		}

		segIndex++
	}

	logrus.Info("Completed to upload file")

	return nil
}

func (uploader *Uploader) waitForFinality(root common.Hash) error {
	logrus.WithField("root", root).Info("Wait for transaction finalized on storage node")

	for {
		time.Sleep(time.Second)

		info, err := uploader.client.GetFileInfo(root)
		if err != nil {
			return errors.WithMessage(err, "Failed to get file info from storage node")
		}

		if info != nil && info.Finalized {
			break
		}
	}

	return nil
}
