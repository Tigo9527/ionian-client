package kv

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"
)

const maxSetSize = 1 << 16 // 64K

var errSizeTooLarge = errors.New("size too large")

type StreamDataBuilder struct {
	version  uint64
	reads    map[common.Hash]map[common.Hash]bool
	writes   map[common.Hash]map[common.Hash][]byte
	controls []AccessControl
}

func NewStreamDataBuilder(version uint64) *StreamDataBuilder {
	return &StreamDataBuilder{
		version: version,
	}
}

func (builder *StreamDataBuilder) Build() (*StreamData, error) {
	if len(builder.controls) > maxSetSize {
		return nil, errSizeTooLarge
	}

	data := StreamData{
		Version:  builder.version,
		Controls: builder.controls,
	}

	for streamId, keys := range builder.reads {
		for k := range keys {
			data.Reads = append(data.Reads, StreamRead{
				StreamId: streamId,
				Key:      k,
			})

			if len(data.Reads) > maxSetSize {
				return nil, errSizeTooLarge
			}
		}
	}

	for streamId, keys := range builder.writes {
		for k, d := range keys {
			data.Writes = append(data.Writes, StreamWrite{
				StreamId: streamId,
				Key:      k,
				Data:     d,
			})

			if len(data.Writes) > maxSetSize {
				return nil, errSizeTooLarge
			}
		}
	}

	return &data, nil
}

func (builder *StreamDataBuilder) WithRead(StreamId, key common.Hash) *StreamDataBuilder {
	if keys, ok := builder.reads[StreamId]; ok {
		keys[key] = true
	} else {
		builder.reads[StreamId] = make(map[common.Hash]bool)
		builder.reads[StreamId][key] = true
	}

	return builder
}

func (builder *StreamDataBuilder) WithWrite(StreamId, key common.Hash, data []byte) *StreamDataBuilder {
	if keys, ok := builder.writes[StreamId]; ok {
		keys[key] = data
	} else {
		builder.writes[StreamId] = make(map[common.Hash][]byte)
		builder.writes[StreamId][key] = data
	}

	return builder
}

func (builder *StreamDataBuilder) withControl(t AccessControlType, streamId common.Hash, account *common.Address, key *common.Hash) *StreamDataBuilder {
	builder.controls = append(builder.controls, AccessControl{
		Type:     t,
		StreamId: streamId,
		Account:  account,
		Key:      key,
	})

	return builder
}

func (builder *StreamDataBuilder) WithControlGrantAdminRole(streamId common.Hash, account common.Address) *StreamDataBuilder {
	return builder.withControl(AclTypeGrantAdminRole, streamId, &account, nil)
}

func (builder *StreamDataBuilder) WithControlRenounceAdminRole(streamId common.Hash) *StreamDataBuilder {
	return builder.withControl(AclTypeRenounceAdminRole, streamId, nil, nil)
}

func (builder *StreamDataBuilder) WithControlSetKeyToSpecial(streamId, key common.Hash) *StreamDataBuilder {
	return builder.withControl(AclTypeSetKeyToSpecial, streamId, nil, &key)
}

func (builder *StreamDataBuilder) WithControlSetKeyToNormal(streamId, key common.Hash) *StreamDataBuilder {
	return builder.withControl(AclTypeSetKeyToNormal, streamId, nil, &key)
}

func (builder *StreamDataBuilder) WithControlGrantWriteRole(streamId common.Hash, account common.Address) *StreamDataBuilder {
	return builder.withControl(AclTypeGrantWriteRole, streamId, &account, nil)
}

func (builder *StreamDataBuilder) WithControlRevokeWriteRole(streamId common.Hash, account common.Address) *StreamDataBuilder {
	return builder.withControl(AclTypeRevokeWriteRole, streamId, &account, nil)
}

func (builder *StreamDataBuilder) WithControlRenounceWriteRole(streamId common.Hash) *StreamDataBuilder {
	return builder.withControl(AclTypeGrantWriteRole, streamId, nil, nil)
}

func (builder *StreamDataBuilder) WithControlGrantSpecialWriteRole(streamId, key common.Hash, account common.Address) *StreamDataBuilder {
	return builder.withControl(AclTypeGrantSpecialWriteRole, streamId, &account, &key)
}

func (builder *StreamDataBuilder) WithControlRevokeSpecialWriteRole(streamId, key common.Hash, account common.Address) *StreamDataBuilder {
	return builder.withControl(AclTypeRevokeSpecialWriteRole, streamId, &account, &key)
}

func (builder *StreamDataBuilder) WithControlRenounceSpecialWriteRole(streamId, key common.Hash) *StreamDataBuilder {
	return builder.withControl(AclTypeRenounceSpecialWriteRole, streamId, nil, &key)
}
