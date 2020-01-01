package protocol

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"

	"github.com/anupcshan/gonbd/blockdevice"
)

const maxSafeOptionLength = 4096

type serverConnection struct {
	conn    net.Conn
	exports map[string]blockdevice.BlockDevice
}

// https://github.com/NetworkBlockDevice/nbd/blob/master/doc/proto.md#fixed-newstyle-negotiation
func (c *serverConnection) negotiate() error {
	header := nbdFixedNewStyleHeader{
		NBDMAGIC,
		IHAVEOPT,
		FlagFixedNewStyle | FlagNoZeroes,
	}
	if err := binary.Write(c.conn, binary.BigEndian, header); err != nil {
		return fmt.Errorf("Error writing header: %w", err)
	}

	var clientFlags nbdClientFlags
	if err := binary.Read(c.conn, binary.BigEndian, &clientFlags); err != nil {
		return fmt.Errorf("Error reading client flags: %w", err)
	}

	if clientFlags&FlagClientFixedNewStyle != FlagClientFixedNewStyle {
		log.Println("WARN: Client flags did not set NBD_FLAG_C_FIXED_NEWSTYLE")
	}

	// Option haggling
	for {
		var clientOptions nbdClientOptions
		if err := binary.Read(c.conn, binary.BigEndian, &clientOptions); err != nil {
			return fmt.Errorf("Error reading client options: %w", err)
		}

		if clientOptions.OptMagic != IHAVEOPT {
			return fmt.Errorf("Bad client option magic %x (expected %x)", clientOptions.OptMagic, IHAVEOPT)
		}

		switch clientOptions.Option {
		case OptAbort:
			if err := binary.Write(c.conn, binary.BigEndian, nbdOptReply{REPLYMAGIC, OptAbort, RepAck, 0}); err != nil {
				return fmt.Errorf("Error writing header: %w", err)
			}
			_ = c.conn.Close()
			return fmt.Errorf("Connection aborted")
		case OptExportName:
			if clientOptions.Length > maxSafeOptionLength {
				_ = c.conn.Close()
				return fmt.Errorf("Option length too long: %d > %d", clientOptions.Length, maxSafeOptionLength)
			}
			exportName := make([]byte, clientOptions.Length)
			if _, err := io.ReadFull(c.conn, exportName); err != nil {
				_ = c.conn.Close()
				return fmt.Errorf("Error reading export name: %w", err)
			}

			export, ok := c.exports[string(exportName)]
			if !ok {
				_ = c.conn.Close()
				return fmt.Errorf("Unknown export %s", exportName)
			}
			return fmt.Errorf("TODO: successful export name not implemented %v", export)
		}
	}
}
