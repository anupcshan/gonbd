package protocol

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"sort"

	"github.com/anupcshan/gonbd/blockdevice"
)

const maxSafeOptionLength = 4096

type nbdServer struct {
	exports map[string]blockdevice.BlockDevice
}

func (c *nbdServer) Serve(ctx context.Context, l net.Listener) error {
	for {
		conn, err := l.Accept()
		if err != nil {
			return err
		}

		go c.handleConnection(conn)
	}
}

type connParams struct {
	export blockdevice.BlockDevice
}

func (c *nbdServer) handleConnection(conn net.Conn) error {
	params, err := c.negotiate(conn)
	if err != nil {
		return err
	}

	return c.handleTransmission(params, conn)
}

func (c *nbdServer) handleTransmission(params *connParams, conn net.Conn) error {
	panic("Not implemented")
}

// https://github.com/NetworkBlockDevice/nbd/blob/master/doc/proto.md#fixed-newstyle-negotiation
func (c *nbdServer) negotiate(conn net.Conn) (*connParams, error) {
	header := nbdFixedNewStyleHeader{
		NBDMAGIC,
		IHAVEOPT,
		FlagFixedNewStyle | FlagNoZeroes,
	}
	if err := binary.Write(conn, binary.BigEndian, header); err != nil {
		return nil, fmt.Errorf("Error writing header: %w", err)
	}

	var clientFlags nbdClientFlags
	if err := binary.Read(conn, binary.BigEndian, &clientFlags); err != nil {
		return nil, fmt.Errorf("Error reading client flags: %w", err)
	}

	if clientFlags&FlagClientFixedNewStyle != FlagClientFixedNewStyle {
		log.Println("WARN: Client flags did not set NBD_FLAG_C_FIXED_NEWSTYLE")
	}

	// Option haggling
	for {
		var clientOptions nbdClientOptions
		if err := binary.Read(conn, binary.BigEndian, &clientOptions); err != nil {
			return nil, fmt.Errorf("Error reading client options: %w", err)
		}

		if clientOptions.OptMagic != IHAVEOPT {
			return nil, fmt.Errorf("Bad client option magic %x (expected %x)", clientOptions.OptMagic, IHAVEOPT)
		}

		switch clientOptions.Option {
		case OptExportName:
			if clientOptions.Length > maxSafeOptionLength {
				_ = conn.Close()
				return nil, fmt.Errorf("Option length too long: %d > %d", clientOptions.Length, maxSafeOptionLength)
			}
			exportName := make([]byte, clientOptions.Length)
			if _, err := io.ReadFull(conn, exportName); err != nil {
				_ = conn.Close()
				return nil, fmt.Errorf("Error reading export name: %w", err)
			}

			export, ok := c.exports[string(exportName)]
			if !ok {
				_ = conn.Close()
				return nil, fmt.Errorf("Unknown export %s", exportName)
			}
			return &connParams{
				export: export,
			}, nil

		case OptAbort:
			if err := binary.Write(conn, binary.BigEndian, nbdOptReply{REPLYMAGIC, clientOptions.Option, RepAck, 0}); err != nil {
				return nil, fmt.Errorf("Error writing header: %w", err)
			}
			_ = conn.Close()
			return nil, fmt.Errorf("Connection aborted")

		case OptList:
			exportNames := make([]string, 0, len(c.exports))
			for k := range c.exports {
				exportNames = append(exportNames, k)
			}
			sort.Strings(exportNames)
			for _, exportName := range exportNames {
				if err := binary.Write(conn, binary.BigEndian, nbdOptReply{REPLYMAGIC, clientOptions.Option, RepServer, 4 + uint32(len(exportName))}); err != nil {
					return nil, fmt.Errorf("Error writing header: %w", err)
				}
				if err := binary.Write(conn, binary.BigEndian, uint32(len(exportName))); err != nil {
					return nil, fmt.Errorf("Error writing header: %w", err)
				}
				if _, err := conn.Write([]byte(exportName)); err != nil {
					return nil, fmt.Errorf("Error writing exportName: %w", err)
				}
			}
			if err := binary.Write(conn, binary.BigEndian, nbdOptReply{REPLYMAGIC, clientOptions.Option, RepAck, 0}); err != nil {
				return nil, fmt.Errorf("Error writing header: %w", err)
			}
		}
	}
}
