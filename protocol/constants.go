package protocol

// NBD Magic constants
const (
	NBDMAGIC   = 0x4e42444d41474943
	IHAVEOPT   = 0x49484156454F5054
	REPLYMAGIC = 0x3e889045565a9
)

// Handshake flags
// https://github.com/NetworkBlockDevice/nbd/blob/master/doc/proto.md#handshake-flags
const (
	FlagFixedNewStyle = 1 << 0
	FlagNoZeroes      = 1 << 1
)

// Client flags
// https://github.com/NetworkBlockDevice/nbd/blob/master/doc/proto.md#client-flags
const (
	FlagClientFixedNewStyle = 1 << 0
	FlagClientNoZeroes      = 1 << 1
)

// Option types
// https://github.com/NetworkBlockDevice/nbd/blob/master/doc/proto.md#option-types
const (
	OptExportName      = 1
	OptAbort           = 2
	OptList            = 3
	OptPeekExport      = 4
	OptStartTLS        = 5
	OptInfo            = 6
	OptGo              = 7
	OptStructuredReply = 8
	OptListMetaContext = 9
	OptSetMetaContext  = 10
)

// Option reply types
// https://github.com/NetworkBlockDevice/nbd/blob/master/doc/proto.md#option-reply-types
const (
	RepAck              = 1
	RepServer           = 2
	RepInfo             = 3
	RepMetaContext      = 4
	RepErrUnsup         = 1<<31 + 1
	RepErrPolicy        = 1<<31 + 2
	RepErrInvalid       = 1<<31 + 3
	RepErrPlatform      = 1<<31 + 4
	RepErrTLSReqd       = 1<<31 + 5
	RepErrUnknown       = 1<<31 + 6
	RepErrShutdown      = 1<<31 + 7
	RepErrBlockSizeReqd = 1<<31 + 8
	RepErrTooBig        = 1<<31 + 9
)

// NBD fixed newstyle server header
// https://github.com/NetworkBlockDevice/nbd/blob/master/doc/proto.md#newstyle-negotiation
// https://github.com/NetworkBlockDevice/nbd/blob/master/doc/proto.md#fixed-newstyle-negotiation
type nbdFixedNewStyleHeader struct {
	magic          uint64
	differentMagic uint64
	handshakeFlags uint16
}

// NBD newstyle client flags
// https://github.com/NetworkBlockDevice/nbd/blob/master/doc/proto.md#newstyle-negotiation
type nbdClientFlags uint32

type nbdClientOptions struct {
	OptMagic uint64
	Option   uint32
	Length   uint32
}

type nbdOptReply struct {
	ReplyMagic   uint64
	ClientOption uint32
	ReplyType    uint32
	Length       uint32
}
