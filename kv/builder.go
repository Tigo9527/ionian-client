package kv

import (
	"errors"
	"sort"

	"github.com/ethereum/go-ethereum/common"
)

const maxSetSize = 1 << 16 // 64K

var errSizeTooLarge = errors.New("size too large")

type builder struct {
	streamIds map[common.Hash]bool // to build tags
}

func (builder *builder) AddStreamId(streamId common.Hash) {
	builder.streamIds[streamId] = true
}

func (builder *builder) BuildTags(sorted ...bool) []byte {
	var ids []common.Hash

	for k := range builder.streamIds {
		ids = append(ids, k)
	}

	if len(sorted) > 0 {
		if sorted[0] {
			sort.SliceStable(ids, func(i, j int) bool {
				return ids[i].Hex() < ids[j].Hex()
			})
		}
	}

	return CreateTags(ids...)
}

type StreamDataBuilder struct {
	AccessControlBuilder
	version uint64
	reads   map[common.Hash]map[common.Hash]bool
	writes  map[common.Hash]map[common.Hash][]byte
}

func NewStreamDataBuilder(version uint64) *StreamDataBuilder {
	return &StreamDataBuilder{
		AccessControlBuilder: AccessControlBuilder{
			builder: builder{
				streamIds: make(map[common.Hash]bool),
			},
			controls: make([]AccessControl, 0),
		},
		version: version,
		reads:   make(map[common.Hash]map[common.Hash]bool),
		writes:  make(map[common.Hash]map[common.Hash][]byte),
	}
}

func (builder *StreamDataBuilder) Build(sorted ...bool) (*StreamData, error) {
	var err error
	data := StreamData{
		Version: builder.version,
	}

	// controls
	if data.Controls, err = builder.AccessControlBuilder.Build(); err != nil {
		return nil, err
	}

	// reads
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

	// writes
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

	if len(sorted) > 0 {
		if sorted[0] {
			sort.SliceStable(data.Reads, func(i, j int) bool {
				streamIdI := data.Reads[i].StreamId.Hex()
				streamIdJ := data.Reads[j].StreamId.Hex()
				if streamIdI == streamIdJ {
					return data.Reads[i].Key.Hex() < data.Reads[j].Key.Hex()
				} else {
					return streamIdI < streamIdJ
				}
			})
			sort.SliceStable(data.Writes, func(i, j int) bool {
				streamIdI := data.Writes[i].StreamId.Hex()
				streamIdJ := data.Writes[j].StreamId.Hex()
				if streamIdI == streamIdJ {
					return data.Writes[i].Key.Hex() < data.Writes[j].Key.Hex()
				} else {
					return streamIdI < streamIdJ
				}
			})
		}
	}

	return &data, nil
}

func (builder *StreamDataBuilder) SetVersion(version uint64) *StreamDataBuilder {
	builder.version = version
	return builder
}

func (builder *StreamDataBuilder) Watch(streamId, key common.Hash) *StreamDataBuilder {
	if keys, ok := builder.reads[streamId]; ok {
		keys[key] = true
	} else {
		builder.reads[streamId] = make(map[common.Hash]bool)
		builder.reads[streamId][key] = true
	}

	return builder
}

func (builder *StreamDataBuilder) Set(streamId, key common.Hash, data []byte) *StreamDataBuilder {
	builder.AddStreamId(streamId)

	if keys, ok := builder.writes[streamId]; ok {
		keys[key] = data
	} else {
		builder.writes[streamId] = make(map[common.Hash][]byte)
		builder.writes[streamId][key] = data
	}

	return builder
}

type AccessControlBuilder struct {
	builder
	controls []AccessControl
}

func (builder *AccessControlBuilder) Build() ([]AccessControl, error) {
	if len(builder.controls) > maxSetSize {
		return nil, errSizeTooLarge
	}

	return builder.controls, nil
}

func (builder *AccessControlBuilder) withControl(t AccessControlType, streamId common.Hash, account *common.Address, key *common.Hash) *AccessControlBuilder {
	builder.AddStreamId(streamId)

	builder.controls = append(builder.controls, AccessControl{
		Type:     t,
		StreamId: streamId,
		Account:  account,
		Key:      key,
	})

	return builder
}

func (builder *AccessControlBuilder) GrantAdminRole(streamId common.Hash, account common.Address) *AccessControlBuilder {
	return builder.withControl(AclTypeGrantAdminRole, streamId, &account, nil)
}

func (builder *AccessControlBuilder) RenounceAdminRole(streamId common.Hash) *AccessControlBuilder {
	return builder.withControl(AclTypeRenounceAdminRole, streamId, nil, nil)
}

func (builder *AccessControlBuilder) SetKeyToSpecial(streamId, key common.Hash) *AccessControlBuilder {
	return builder.withControl(AclTypeSetKeyToSpecial, streamId, nil, &key)
}

func (builder *AccessControlBuilder) SetKeyToNormal(streamId, key common.Hash) *AccessControlBuilder {
	return builder.withControl(AclTypeSetKeyToNormal, streamId, nil, &key)
}

func (builder *AccessControlBuilder) GrantWriteRole(streamId common.Hash, account common.Address) *AccessControlBuilder {
	return builder.withControl(AclTypeGrantWriteRole, streamId, &account, nil)
}

func (builder *AccessControlBuilder) RevokeWriteRole(streamId common.Hash, account common.Address) *AccessControlBuilder {
	return builder.withControl(AclTypeRevokeWriteRole, streamId, &account, nil)
}

func (builder *AccessControlBuilder) RenounceWriteRole(streamId common.Hash) *AccessControlBuilder {
	return builder.withControl(AclTypeRenounceWriteRole, streamId, nil, nil)
}

func (builder *AccessControlBuilder) GrantSpecialWriteRole(streamId, key common.Hash, account common.Address) *AccessControlBuilder {
	return builder.withControl(AclTypeGrantSpecialWriteRole, streamId, &account, &key)
}

func (builder *AccessControlBuilder) RevokeSpecialWriteRole(streamId, key common.Hash, account common.Address) *AccessControlBuilder {
	return builder.withControl(AclTypeRevokeSpecialWriteRole, streamId, &account, &key)
}

func (builder *AccessControlBuilder) RenounceSpecialWriteRole(streamId, key common.Hash) *AccessControlBuilder {
	return builder.withControl(AclTypeRenounceSpecialWriteRole, streamId, nil, &key)
}
