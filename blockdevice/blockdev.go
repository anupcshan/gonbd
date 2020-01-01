package blockdevice

// WriteFlags holds options/flags passed into a write-like operation.
type WriteFlags int8

const (
	requestFUA WriteFlags = 1 << iota
)

// FUA indicates if this write-like operation requested FUA.
func (wo WriteFlags) FUA() bool {
	return wo&requestFUA != 0
}

// BlockDeviceFeatures holds a set of features this block device supports (or flags describing this
// block device).
type BlockDeviceFeatures int16

const (
	// SupportsFUA indicates this block device supports NBD_CMD_FLAG_FUA.
	SupportsFUA BlockDeviceFeatures = 1 << iota
	// IsRotational sets NBD_FLAG_ROTATIONAL, indicating this block device exports the characteristics
	// of a rotational medium. Clients may choose to use elevator algorithm to interact with this
	// device.
	IsRotational
)

// BlockDevice is a minimal block device interface. Each NBD backend must implement this interface.
type BlockDevice interface {
	WriteAt(p []byte, offset uint64, opts WriteFlags) (uint64, error)
	ReadAt(p []byte, offset uint64) (uint64, error)
	Features() BlockDeviceFeatures
	Size() uint64

	// Optional:
	// BlockDeviceFlusher
	// BlockDeviceTrimmer
	// BlockDeviceCacher
	// BlockDeviceFastWriteZeroer
	// BlockDeviceSizeConstraints
}

// BlockDeviceFlusher indicates this BlockDevice supports NBD_CMD_FLUSH.
type BlockDeviceFlusher interface {
	Flush() error
}

// BlockDeviceTrimmer indicates this BlockDevice supports NBD_CMD_TRIM.
type BlockDeviceTrimmer interface {
	TrimAt(offset uint64, length uint64, opts WriteFlags) (uint64, error)
}

// BlockDeviceCacher indicates this BlockDevice supports NBD_CMD_CACHE.
type BlockDeviceCacher interface {
	CacheAt(offset uint64, length uint64) (uint64, error)
}

// BlockDeviceFastWriteZeroer indicates this BlockDevice supports NBD_FLAG_SEND_FAST_ZERO.
// By default, the server implementation will accept NBD_CMD_WRITE_ZEROES and translate them to
// WriteAt operations on the block device (or TrimAt, if supported). This interface indicates the
// block device supports faster zeroing out of byte ranges (say, in a sparse representation).
type BlockDeviceFastWriteZeroer interface {
	FastWriteZeroes(offset uint64, length uint64, opts WriteFlags) (uint64, error)
}

// BlockDeviceSizeConstraints indicates this BlockDevice has specific size constraints which the
// client must send NBD_INFO_BLOCK_SIZE during option haggling - if not, NBD_REP_ERR_BLOCK_SIZE_REQD
// will be sent back on NBD_OPT_EXPORT_NAME.
type BlockDeviceSizeConstraints interface {
	SizeConstaints() (min uint64, preferred uint64, max uint64)
}
