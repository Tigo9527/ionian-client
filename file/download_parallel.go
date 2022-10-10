package file

import (
	"fmt"

	"github.com/Ionian-Web3-Storage/ionian-client/common/parallel"
	"github.com/Ionian-Web3-Storage/ionian-client/file/download"
	"github.com/Ionian-Web3-Storage/ionian-client/node"
	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const minBufSize = 8

type SegmentDownloader struct {
	clients []*node.Client
	file    *download.DownloadingFile

	withProof bool

	segmentOffset uint32
	numChunks     uint32
	numSegments   uint32
}

func NewSegmentDownloader(clients []*node.Client, file *download.DownloadingFile, withProof bool) (*SegmentDownloader, error) {
	offset := file.Metadata().Offset
	if offset%DefaultSegmentSize > 0 {
		return nil, errors.Errorf("Invalid data offset in downloading file %v", offset)
	}

	fileSize := file.Metadata().Size

	return &SegmentDownloader{
		clients: clients,
		file:    file,

		withProof: withProof,

		segmentOffset: uint32(offset / DefaultSegmentSize),
		numChunks:     numSplits(fileSize, DefaultChunkSize),
		numSegments:   numSplits(fileSize, DefaultSegmentSize),
	}, nil
}

// Download downloads segments in parallel.
func (downloader *SegmentDownloader) Download() error {
	numTasks := downloader.numSegments - downloader.segmentOffset
	numNodes := len(downloader.clients)
	bufSize := numNodes * 2
	if bufSize < minBufSize {
		bufSize = minBufSize
	}

	return parallel.Serial(downloader, int(numTasks), numNodes, bufSize)
}

// ParallelDo implements the parallel.Interface interface.
func (downloader *SegmentDownloader) ParallelDo(routine, task int) (interface{}, error) {
	segmentIndex := downloader.segmentOffset + uint32(task)
	startIndex := segmentIndex * DefaultSegmentMaxChunks
	endIndex := startIndex + DefaultSegmentMaxChunks
	if endIndex > downloader.numChunks {
		endIndex = downloader.numChunks
	}

	root := downloader.file.Metadata().Root

	var (
		segment []byte
		err     error
	)

	if downloader.withProof {
		segment, err = downloader.downloadWithProof(downloader.clients[routine], root, startIndex, endIndex)
	} else {
		segment, err = downloader.clients[routine].DownloadSegment(root, startIndex, endIndex)
	}

	if err != nil {
		logrus.WithError(err).WithFields(logrus.Fields{
			"routine": routine,
			"segment": fmt.Sprintf("%v/%v", segmentIndex, downloader.numSegments),
			"chunks":  fmt.Sprintf("[%v, %v)", startIndex, endIndex),
		}).Error("Failed to download segment")
	} else if logrus.IsLevelEnabled(logrus.TraceLevel) {
		logrus.WithFields(logrus.Fields{
			"routine": routine,
			"segment": fmt.Sprintf("%v/%v", segmentIndex, downloader.numSegments),
			"chunks":  fmt.Sprintf("[%v, %v)", startIndex, endIndex),
		}).Trace("Succeeded to download segment")
	}

	// remove paddings for the last chunk
	if err == nil && segmentIndex == downloader.numSegments-1 {
		fileSize := downloader.file.Metadata().Size
		if lastChunkSize := fileSize % DefaultChunkSize; lastChunkSize > 0 {
			paddings := DefaultChunkSize - lastChunkSize
			segment = segment[0 : len(segment)-int(paddings)]
		}
	}

	return segment, err
}

// ParallelCollect implements the parallel.Interface interface.
func (downloader *SegmentDownloader) ParallelCollect(result *parallel.Result) error {
	return downloader.file.Write(result.Value.([]byte))
}

func (downloader *SegmentDownloader) downloadWithProof(client *node.Client, root common.Hash, startIndex, endIndex uint32) ([]byte, error) {
	segmentIndex := startIndex / uint32(DefaultSegmentMaxChunks)

	segment, err := client.DownloadSegmentWithProof(root, segmentIndex)
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to download segment with proof from storage node")
	}

	if expectedDataLen := int(endIndex-startIndex) * DefaultChunkSize; expectedDataLen != len(segment.Data) {
		return nil, errors.Errorf("Downloaded data length mismatch, expected = %v, actual = %v", expectedDataLen, len(segment.Data))
	}

	numChunksFlowPadded, _ := computePaddedSize(downloader.numChunks)
	numSegmentsFlowPadded := (numChunksFlowPadded-1)/DefaultSegmentMaxChunks + 1

	// pad empty chunks for the last segment to validate merkle proof
	var emptyChunksPadded uint32
	if numChunks := endIndex - startIndex; numChunks < DefaultSegmentMaxChunks {
		if uint64(segmentIndex) < numSegmentsFlowPadded-1 {
			// pad empty chunks to a full segment
			emptyChunksPadded = DefaultSegmentMaxChunks - numChunks
		} else if lastSegmentChunks := uint32(numChunksFlowPadded % DefaultSegmentMaxChunks); numChunks < lastSegmentChunks {
			// pad for the last segment with flow padded empty chunks
			emptyChunksPadded = lastSegmentChunks - numChunks
		}
	}

	segmentRootHash := segmentRoot(segment.Data, emptyChunksPadded)

	if err := segment.Proof.ValidateHash(root, segmentRootHash, segmentIndex, uint32(numSegmentsFlowPadded)); err != nil {
		return nil, errors.WithMessage(err, "Failed to validate proof")
	}

	return segment.Data, nil
}
