package file

import (
	"errors"
	"os"

	"github.com/Ionian-Web3-Storage/ionian-client/file/merkle"
	"github.com/ethereum/go-ethereum/common"
)

const (
	// DefaultChunkSize represents the default chunk size in bytes.
	DefaultChunkSize = 256

	// DefaultSegmentMaxChunks represents the default maximum number of chunks within a segment.
	DefaultSegmentMaxChunks = 1024

	// DefaultSegmentSize represents the default segment size in bytes.
	DefaultSegmentSize = DefaultChunkSize * DefaultSegmentMaxChunks
)

var (
	// ErrFileRequired is returned when manipulate on a folder.
	ErrFileRequired = errors.New("file required")

	// ErrFileEmpty is returned when empty file opened.
	ErrFileEmpty = errors.New("file is empty")
)

type File struct {
	os.FileInfo
	underlying *os.File
}

func Exists(name string) (bool, error) {
	file, err := os.Open(name)
	if os.IsNotExist(err) {
		return false, nil
	}

	if err != nil {
		return false, err
	}

	defer file.Close()

	return true, nil
}

func Open(name string) (*File, error) {
	file, err := os.Open(name)
	if err != nil {
		return nil, err
	}

	info, err := file.Stat()
	if err != nil {
		return nil, err
	}

	if info.IsDir() {
		return nil, ErrFileRequired
	}

	if info.Size() == 0 {
		return nil, ErrFileEmpty
	}

	return &File{
		FileInfo:   info,
		underlying: file,
	}, nil
}

func (file *File) Close() error {
	return file.underlying.Close()
}

func (file *File) NumChunks() uint32 {
	return numSplits(file.Size(), DefaultChunkSize)
}

func (file *File) NumSegments() uint32 {
	return numSplits(file.Size(), DefaultSegmentSize)
}

func (file *File) Iterate(flowPadding bool) *Iterator {
	// File root and the Flow submission has different ways in file padding
	return NewSegmentIterator(file.underlying, file.Size(), 0, flowPadding)
}

func (file *File) MerkleTree() (*merkle.Tree, error) {
	iter := file.Iterate(true)
	var builder merkle.TreeBuilder

	for {
		ok, err := iter.Next()
		if err != nil {
			return nil, err
		}

		if !ok {
			break
		}

		segRoot := segmentRoot(iter.Current())

		builder.AppendHash(segRoot)
	}

	return builder.Build(), nil
}

func numSplits(total int64, unit int) uint32 {
	return uint32((total-1)/int64(unit) + 1)
}

func segmentRoot(chunks []byte) common.Hash {
	dataLen := len(chunks)
	if dataLen == 0 {
		return common.Hash{}
	}

	var builder merkle.TreeBuilder

	for offset := 0; offset < dataLen; offset += DefaultChunkSize {
		chunk := chunks[offset : offset+DefaultChunkSize]
		builder.Append(chunk)
	}

	return builder.Build().Root()
}
