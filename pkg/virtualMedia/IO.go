package virtualMedia

import (
	"github.com/fanap-infra/archiverMedia/pkg/media"
	"google.golang.org/protobuf/proto"
)

func (vm *VirtualMedia) WriteFrame(frame *media.Packet) error {
	if vm.readOnly {
		return ErrFileIsReadOnly
	}
	vm.fwMUX.Lock()
	defer vm.fwMUX.Unlock()
	if vm.frameChunk.Packets == nil {
		vm.log.Warnv("frame chunk is nil", "name", vm.name)
		vm.frameChunk.Packets = []*media.Packet{}
	}

	if frame.PacketType == media.PacketType_PacketVideo && frame.IsKeyFrame {
		if len(vm.frameChunk.Packets) >= FrameChunkMinimumFrameCount {
			// vm.log.Infov("packet chunk is written", "Index", vm.frameChunk.Index,
			//	"packets number", len(vm.frameChunk.Packets), "StartTime", vm.frameChunk.StartTime,
			//	"EndTime", vm.frameChunk.EndTime)
			b, err := generateFrameChunk(vm.frameChunk)
			if err != nil {
				return err
			}
			l, err := vm.vFile.Write(b)
			vm.fileSize += uint32(l)
			if err != nil {
				return err
			}
			if vm.frameChunk.Index == 1 {
				vm.info.StartTime = vm.frameChunk.StartTime
			}
			vm.info.EndTime = vm.frameChunk.EndTime

			err = vm.UpdateFileOptionalData()
			if err != nil {
				return err
			}

			vm.frameChunk = &media.PacketChunk{
				Index: vm.frameChunk.Index + 1, StartTime: vm.frameChunk.EndTime,
				Packets: []*media.Packet{}, PreviousChunkSize: uint32(len(b)),
				PreviousChunkStartAddress: vm.fileSize - uint32(len(b)),
			}
			// vm.log.Infov("Empty packet chunk is written", "Index", vm.frameChunk.Index,
			//	"packets number", len(vm.frameChunk.Packets), "StartTime", vm.frameChunk.StartTime,
			//	"EndTime", vm.frameChunk.EndTime,"size frame chunk", len(b))
		}
	}

	if frame.PacketType == media.PacketType_PacketVideo {
		if frame.Time < 0 || frame.Time < vm.frameChunk.StartTime {
			vm.log.Warnv("frame time is wrong", "frame.Time ",frame.Time ,
				"vm.frameChunk.StartTime",vm.frameChunk.StartTime, "fileID", vm.fileID)
			return nil
		}
		if len(vm.frameChunk.Packets) == 0 && vm.frameChunk.Index == 1 {
			vm.frameChunk.StartTime = frame.Time
		}
		vm.frameChunk.EndTime = frame.Time
	}
	vm.frameChunk.Packets = append(vm.frameChunk.Packets, frame)
	return nil
}

func (vm *VirtualMedia) UpdateFileOptionalData()  error {
	bInfo, err := proto.Marshal(vm.info)
	if err != nil {
		return err
	}
	err = vm.vFile.UpdateFileOptionalData(bInfo)
	if err != nil {
		return err
	}
	return nil
}

func (vm *VirtualMedia) ReadFrame() (*media.Packet, error) {
	vm.rxMUX.Lock()
	defer vm.rxMUX.Unlock()
	if vm.frameChunkRX == nil || int(vm.currentFrameInChunk) >= len(vm.frameChunkRX.Packets) {
		fc, err := vm.NextFrameChunk()
		if err != nil {
			vm.log.Warnv("can not get next frame chunk",
				"currentFrameInChunk", vm.currentFrameInChunk)
			return nil, err
		}
		vm.frameChunkRX = fc
		vm.currentFrameInChunk = 0
	} else if uint32(len(vm.frameChunkRX.Packets)) <= (vm.currentFrameInChunk) {
		fc, err := vm.NextFrameChunk()
		if err != nil {
			// vm.log.Warnv("can not get next frame chunk",
			//	"frame chunk index", vm.frameChunkRX.Index, "currentFrameInChunk", vm.currentFrameInChunk)
			return nil, err
		}
		vm.frameChunkRX = fc
		vm.currentFrameInChunk = 0
	}
	vm.currentFrameInChunk++
	return vm.frameChunkRX.Packets[vm.currentFrameInChunk-1], nil
}

func (vm *VirtualMedia) GotoTime(frameTime int64) (int64, error) {
	vm.rxMUX.Lock()
	defer vm.rxMUX.Unlock()
	if vm.frameChunkRX != nil {
		if vm.frameChunkRX.StartTime <= frameTime &&
			vm.frameChunkRX.EndTime >= frameTime {
			vm.currentFrameInChunk = 0
			return vm.frameChunkRX.StartTime, nil
		}
	}
	approximateByteIndex := int64(0)
	if vm.info.EndTime - vm.info.StartTime != 0 {
		approximateByteIndex = frameTime * int64(vm.vFile.GetFileSize()) / (vm.info.EndTime - vm.info.StartTime)
	} else {
		vm.log.Warnv("virtual media info time are wrong", "fileID",vm.fileID,
			"endTime",vm.info.EndTime, "StartTime", vm.info.StartTime )
	}

	vm.vfBuf = vm.vfBuf[:0]
	// moving forward is easier than backward moving
	approximateByteIndex = approximateByteIndex - int64(vm.blockSize*2)
	if approximateByteIndex < 0 {
		approximateByteIndex = 0
	}
	err := vm.vFile.ChangeSeekPointer(approximateByteIndex)
	if err != nil {
		return 0, err
	}

	_, err = vm.NextFrameChunk()
	if err != nil {
		return 0, err
	}
	//if vm.frameChunkRX == nil {
	//	// tmpBuf := make([]byte, 2*vm.blockSize)
	//
	//	//for {
	//	//	n, err := vm.vFile.ReadAt(tmpBuf, int64(seekPointer))
	//	//	if n == 0 {
	//	//		return nil, err
	//	//}
	//}

	for {
		if vm.frameChunkRX.StartTime <= frameTime &&
			vm.frameChunkRX.EndTime >= frameTime {
			return vm.frameChunkRX.StartTime, nil
		} else if vm.frameChunkRX.EndTime < frameTime {
			_, err := vm.NextFrameChunk()
			if err != nil {
				return 0, err
			}
		} else {
			_, err := vm.PreviousFrameChunk()
			if err != nil {
				return 0, err
			}
		}
	}
}

func (vm *VirtualMedia) Close() error {
	vm.fwMUX.Lock()
	defer vm.fwMUX.Unlock()
	vm.rxMUX.Lock()
	defer vm.rxMUX.Unlock()
	if vm.frameChunk != nil {
		if len(vm.frameChunk.Packets) > 0 && !vm.readOnly {
			b, err := generateFrameChunk(vm.frameChunk)
			if err != nil {
			return err
		}
			l, err := vm.vFile.Write(b)
			vm.fileSize += uint32(l)
			if err != nil {
			return err
		}
			if vm.frameChunk.Index == 1 {
				vm.info.StartTime = vm.frameChunk.StartTime
			}
			vm.info.EndTime = vm.frameChunk.EndTime
			err = vm.UpdateFileOptionalData()
			if err != nil {
			return err
		}
		// vm.log.Infov("packet chunk is written in close", "Index", vm.frameChunk.Index,
		//	"packets number", len(vm.frameChunk.Packets), "size frame chunk", len(b))
			vm.frameChunk = &media.PacketChunk{Index: vm.frameChunk.Index + 1}
		}
	} else {
		vm.log.Warnv("virtual file close, frameChunk is nill", "id", vm.fileID)
	}
	// vm.log.Infov("virtual file closed", "vm.frameChunk.Index", vm.frameChunk.Index,
	//	"start time", vm.info.StartTime, "end time", vm.info.EndTime)
	err := vm.vFile.Close()
	if err != nil {
		vm.log.Errorv("virtual media can not close", "err", err.Error())
	}
	vm.vfBuf = vm.vfBuf[:0]
	vm.frameChunkRX = nil
	vm.frameChunk = nil
	return vm.archiver.Closed(vm.fileID)
}

func (vm *VirtualMedia) CloseWithNotifyArchiver() error {
	vm.fwMUX.Lock()
	defer vm.fwMUX.Unlock()
	vm.rxMUX.Lock()
	defer vm.rxMUX.Unlock()

	if len(vm.frameChunk.Packets) > 0 {
		b, err := generateFrameChunk(vm.frameChunk)
		if err != nil {
			return err
		}
		l, err := vm.vFile.Write(b)
		vm.fileSize += uint32(l)
		if err != nil {
			return err
		}
		if vm.frameChunk.Index == 1 {
			vm.info.StartTime = vm.frameChunk.StartTime
		}
		vm.info.EndTime = vm.frameChunk.EndTime
		bInfo, err := proto.Marshal(vm.info)
		if err != nil {
			return err
		}
		err = vm.vFile.UpdateFileOptionalData(bInfo)
		if err != nil {
			return err
		}

		vm.frameChunk = &media.PacketChunk{Index: vm.frameChunk.Index + 1}
	}

	err := vm.vFile.Close()
	if err != nil {
		vm.log.Errorv("virtual media can not close", "err", err.Error())
	}
	vm.vfBuf = vm.vfBuf[:0]
	vm.frameChunkRX = nil
	vm.frameChunk = nil
	return nil
}
