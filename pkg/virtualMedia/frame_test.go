package virtualMedia

import (
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"

	"github.com/fanap-infra/archiverMedia/pkg/media"
)

func TestFrame_Generation(t *testing.T) {
	fc_Index := uint32(5)
	pkt := &media.PacketChunk{Index: fc_Index}
	data, err := generateFrameChunk(pkt)
	assert.Equal(t, nil, err)
	fc := &media.PacketChunk{}
	frameChunkDataSize := binary.BigEndian.Uint32(
		data[FrameChunkIdentifierSize:FrameChunkHeader])
	err = proto.Unmarshal(data[FrameChunkHeader:FrameChunkHeader+frameChunkDataSize], fc)
	assert.Equal(t, nil, err)
	assert.Equal(t, fc_Index, fc.Index)
}
