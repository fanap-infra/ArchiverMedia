package archiverMedia

import (
	"math/rand"
	"os"
	"testing"

	"github.com/fanap-infra/archiverMedia/pkg/media"
	"github.com/fanap-infra/archiverMedia/pkg/utils"
	"github.com/fanap-infra/archiverMedia/pkg/virtualMedia"
	"github.com/fanap-infra/log"
	"github.com/stretchr/testify/assert"
)

func TestIO_ChangeFrameTime(t *testing.T) {
	homePath, err := os.UserHomeDir()
	assert.Equal(t, nil, err)
	_ = utils.DeleteFile(homePath + "/" + fsPath)
	_ = utils.DeleteFile(homePath + "/" + headerPath)
	eventListener := EventsListener{t: t}
	provider := NewProvider()

	blockSizeTestTemp := 128
	arch, err := provider.CreateFileSystem(homePath, int64(blockSizeTestTemp*128), uint32(blockSizeTestTemp), &eventListener,
		log.GetScope("test"))
	assert.Equal(t, nil, err)
	assert.Equal(t, true, utils.FileExists(homePath+"/"+fsPath))
	assert.Equal(t, true, utils.FileExists(homePath+"/"+headerPath))
	// var packets []*media.Packet

	MaxID := 1000

	MaxByteArraySize := int(float32(blockSizeTestTemp) * 0.1)
	VFSize := int(3.5 * float32(blockSizeTestTemp))
	vfID := uint32(rand.Intn(MaxID))
	vm, err := arch.NewVirtualMediaFile(vfID, "test")
	assert.Equal(t, nil, err)
	size := 0
	packetTime := 0
	for {
		token := make([]byte, uint32(rand.Intn(MaxByteArraySize)))
		m, err := rand.Read(token)
		assert.Equal(t, nil, err)
		pkt := &media.Packet{
			Data: token, PacketType: media.PacketType_PacketVideo, IsKeyFrame: true,
			Time: int64(packetTime),
		}
		packetTime = packetTime + 30
		// packets = append(packets, pkt)
		size = size + m
		err = vm.WriteFrame(pkt)
		assert.Equal(t, nil, err)

		if size > VFSize {
			break
		}
	}

	err = vm.Close()
	assert.Equal(t, nil, err)

	vm2, err := arch.OpenVirtualMediaFile(vfID)
	assert.Equal(t, nil, err)
	numberOfTests := 5
	counter := 0
	for {
		mTime := int64(rand.Intn(packetTime))
		resultTime, err := vm2.GotoTime(mTime)
		assert.Equal(t, nil, err)
		assert.GreaterOrEqual(t, mTime+int64(virtualMedia.FrameChunkMinimumFrameCount*30), resultTime)
		assert.GreaterOrEqual(t, resultTime, mTime-int64(virtualMedia.FrameChunkMinimumFrameCount*30))

		pkt, err := vm2.ReadFrame()
		assert.Equal(t, nil, err)
		assert.GreaterOrEqual(t, mTime+virtualMedia.FrameChunkMinimumFrameCount*30, pkt.Time)
		assert.GreaterOrEqual(t, pkt.Time, mTime-virtualMedia.FrameChunkMinimumFrameCount*30)
		counter++
		if counter == numberOfTests {
			break
		}
	}

	err = vm2.Close()
	assert.Equal(t, nil, err)

	err = arch.Close()
	assert.Equal(t, nil, err)

	_ = utils.DeleteFile(homePath + "/" + fsPath)
	_ = utils.DeleteFile(homePath + "/" + headerPath)
}

func TestIO_PreviousFrameChunk(t *testing.T) {
	homePath, err := os.UserHomeDir()
	assert.Equal(t, nil, err)
	_ = utils.DeleteFile(homePath + "/" + fsPath)
	_ = utils.DeleteFile(homePath + "/" + headerPath)
	eventListener := EventsListener{t: t}
	provider := NewProvider()

	blockSizeTestTemp := 128
	arch, err := provider.CreateFileSystem(homePath, int64(blockSizeTestTemp*128), uint32(blockSizeTestTemp), &eventListener,
		log.GetScope("test"))
	assert.Equal(t, nil, err)
	assert.Equal(t, true, utils.FileExists(homePath+"/"+fsPath))
	assert.Equal(t, true, utils.FileExists(homePath+"/"+headerPath))
	// var packets []*media.Packet

	MaxID := 1000

	MaxByteArraySize := int(float32(blockSizeTestTemp) * 0.1)
	VFSize := int(3.5 * float32(blockSizeTestTemp))
	vfID := uint32(rand.Intn(MaxID))
	vm, err := arch.NewVirtualMediaFile(vfID, "test")
	assert.Equal(t, nil, err)
	size := 0
	packetTime := 0
	for {
		token := make([]byte, uint32(rand.Intn(MaxByteArraySize)))
		m, err := rand.Read(token)
		assert.Equal(t, nil, err)
		pkt := &media.Packet{
			Data: token, PacketType: media.PacketType_PacketVideo, IsKeyFrame: true,
			Time: int64(packetTime),
		}
		packetTime = packetTime + 30
		// packets = append(packets, pkt)
		size = size + m
		err = vm.WriteFrame(pkt)
		assert.Equal(t, nil, err)

		if size > VFSize {
			break
		}
	}

	err = vm.Close()
	assert.Equal(t, nil, err)

	vm2, err := arch.OpenVirtualMediaFile(vfID)
	assert.Equal(t, nil, err)
	// numberOfTests := 5
	// counter := 0
	mTime := int64(packetTime - 30)
	resultTime, err := vm2.GotoTime(mTime)
	assert.Equal(t, nil, err)
	assert.GreaterOrEqual(t, mTime+int64(virtualMedia.FrameChunkMinimumFrameCount*30), resultTime)
	assert.GreaterOrEqual(t, resultTime, mTime-int64(virtualMedia.FrameChunkMinimumFrameCount*30))
	fc, err := vm2.PreviousFrameChunk()
	if assert.Equal(t, nil, err) {
		fcStartTime := fc.StartTime
		IDX := fc.Index
		for {
			fc, err = vm2.PreviousFrameChunk()
			assert.Equal(t, nil, err)
			assert.Equal(t, int(IDX-1), int(fc.Index))
			if !assert.Equal(t, fcStartTime, fc.EndTime) {
				break
			}
			assert.GreaterOrEqual(t, fc.EndTime, fc.StartTime)
			fcStartTime = fc.StartTime
			IDX = fc.Index
			// fcEndTime = fc.EndTime
			if fcStartTime < int64(virtualMedia.FrameChunkMinimumFrameCount*30) {
				break
			}
		}
	}

	err = vm2.Close()
	assert.Equal(t, nil, err)

	err = arch.Close()
	assert.Equal(t, nil, err)

	_ = utils.DeleteFile(homePath + "/" + fsPath)
	_ = utils.DeleteFile(homePath + "/" + headerPath)
}
