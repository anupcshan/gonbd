package file

import (
	"os"

	"github.com/anupcshan/gonbd/blockdevice"
)

type fileDevice struct {
	f *os.File
}

func (f *fileDevice) Features() blockdevice.BlockDeviceFeatures {
	return blockdevice.SupportsFUA
}

func (f *fileDevice) ReadAt(p []byte, offset uint64) (uint64, error) {
	n, err := f.f.ReadAt(p, int64(offset))
	return uint64(n), err
}

func (f *fileDevice) WriteAt(p []byte, offset uint64, flags blockdevice.WriteFlags) (uint64, error) {
	n, err := f.f.WriteAt(p, int64(offset))
	if err != nil {
		return uint64(n), err
	}

	if flags.FUA() {
		err = f.f.Sync()
	}
	return uint64(n), err
}

func (f *fileDevice) Size() (uint64, error) {
	info, err := f.f.Stat()
	if err != nil {
		return 0, err
	}

	return uint64(info.Size()), nil
}

var _ blockdevice.BlockDevice = (*fileDevice)(nil)

func NewFileDevice(f *os.File) *fileDevice {
	return &fileDevice{f}
}
