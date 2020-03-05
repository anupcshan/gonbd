package protocol

import (
	"bytes"
	"io"
	"net"
	"sync"
	"testing"

	"github.com/anupcshan/gonbd/blockdevice"
)

func expectRead(t *testing.T, c net.Conn, s []byte) {
	t.Helper()
	expected := new(bytes.Buffer)
	if _, err := expected.Write(s); err != nil {
		t.Fatal(err)
	}

	actual := make([]byte, expected.Len())
	if _, err := io.ReadFull(c, actual); err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(expected.Bytes(), actual) {
		t.Errorf("Mismatch in bytes read:\nExpected: %03v\nActual:   %03v", expected.Bytes(), actual)
	}
}

func expectWrite(t *testing.T, c net.Conn, s []byte) {
	t.Helper()
	if _, err := c.Write(s); err != nil {
		t.Fatal(err)
	}
}

type opcode int8

const (
	opRead opcode = iota + 1
	opWrite
	opClose
)

type operation struct {
	code opcode
	data []byte
}

type testcase struct {
	desc             string
	exports          map[string]blockdevice.BlockDevice
	ops              []operation
	expectErr        bool
	expectConnClosed bool
}

func runTest(t *testing.T, c testcase) {
	t.Logf("Running case: %s", c.desc)
	serverConn, clientConn := netPipe()
	sc := new(nbdServer)
	sc.exports = c.exports

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()

		for _, op := range c.ops {
			switch op.code {
			case opRead:
				expectRead(t, clientConn, op.data)
			case opWrite:
				expectWrite(t, clientConn, op.data)
			case opClose:
				clientConn.Close()
			}
		}
	}()

	_, err := sc.negotiate(serverConn)
	if c.expectErr && err == nil {
		t.Error("Expected error during negotiation but got no error")
	} else if !c.expectErr && err != nil {
		t.Errorf("Expected no error during negotiation but got error %v", err)
	}

	if c.expectConnClosed && !serverConn.HasClosed() {
		t.Error("Expected connection to be closed, but it is still open")
	}
	if !c.expectConnClosed && serverConn.HasClosed() {
		t.Error("Expected connection to still be open, but it has been closed")
	}
	wg.Wait()
}

// nbdFixedNewStyleHeader{NBDMAGIC, IHAVEOPT, FlagFixedNewStyle | FlagNoZeroes}
var fixedNewStyleHeader = []byte{
	// NBDMAGIC
	0x4e, 0x42, 0x44, 0x4d, 0x41, 0x47, 0x49, 0x43,
	// IHAVEOPT
	0x49, 0x48, 0x41, 0x56, 0x45, 0x4f, 0x50, 0x54,
	// FlagFixedNewStyle | FlagNoZeroes
	0x00, 0x03,
}

// nbdClientFlags(FlagClientFixedNewStyle | FlagClientNoZeroes)
var fixedNewStyleClientHeader = []byte{
	0x00, 0x00, 0x00, 0x03,
}

func TestNegotiation(t *testing.T) {
	for _, c := range []testcase{
		{
			desc: "Don't fail on missing NBD_FLAG_C_FIXED_NEWSTYLE",
			ops: []operation{
				{opRead, fixedNewStyleHeader},
				// nbdClientFlags(0)
				{opWrite, []byte{0x00, 0x00, 0x00, 0x00}},
				{opClose, nil},
			},
			// Server shouldn't have closed connection.
			expectConnClosed: false,
			expectErr:        true, // EOF
		},
		{
			desc: "Handle abort",
			ops: []operation{
				{opRead, fixedNewStyleHeader},
				{opWrite, fixedNewStyleClientHeader},
				// {opWrite, nbdClientOptions{OptMagic: IHAVEOPT, Option: OptAbort}},
				{opWrite, []byte{
					// IHAVEOPT
					0x49, 0x48, 0x41, 0x56, 0x45, 0x4f, 0x50, 0x54,
					// OptAbort
					0x00, 0x00, 0x00, 0x02,
					// Length
					0x00, 0x00, 0x00, 0x00,
				}},
				// nbdOptReply{REPLYMAGIC, OptAbort, RepAck, 0}
				{opRead, []byte{
					// REPLYMAGIC
					0x00, 0x03, 0xe8, 0x89, 0x04, 0x55, 0x65, 0xa9,
					// OptAbort
					0x00, 0x00, 0x00, 0x02,
					// RepAck
					0x00, 0x00, 0x00, 0x01,
					// Length
					0x00, 0x00, 0x00, 0x00,
				}},
			},
			expectConnClosed: true,
			expectErr:        true,
		},
		{
			desc: "Handle list with 0 exports",
			ops: []operation{
				{opRead, fixedNewStyleHeader},
				{opWrite, fixedNewStyleClientHeader},
				// nbdClientOptions{OptMagic: IHAVEOPT, Option: OptList}
				{opWrite, []byte{
					// IHAVEOPT
					0x49, 0x48, 0x41, 0x56, 0x45, 0x4f, 0x50, 0x54,
					// OptList
					0x00, 0x00, 0x00, 0x03,
					// Length
					0x00, 0x00, 0x00, 0x00,
				}},
				// nbdOptReply{REPLYMAGIC, OptList, RepAck, 0}
				{opRead, []byte{
					// REPLYMAGIC
					0x00, 0x03, 0xe8, 0x89, 0x04, 0x55, 0x65, 0xa9,
					// OptList
					0x00, 0x00, 0x00, 0x03,
					// RepAck
					0x00, 0x00, 0x00, 0x01,
					// Length
					0x00, 0x00, 0x00, 0x00,
				}},
				{opClose, nil},
			},
			expectConnClosed: false,
			expectErr:        true, // EOF
		},
		{
			desc: "Handle list with 1 export",
			exports: map[string]blockdevice.BlockDevice{
				"foo": nil,
			},
			ops: []operation{
				{opRead, fixedNewStyleHeader},
				{opWrite, fixedNewStyleClientHeader},
				// nbdClientOptions{OptMagic: IHAVEOPT, Option: OptList}
				{opWrite, []byte{
					// IHAVEOPT
					0x49, 0x48, 0x41, 0x56, 0x45, 0x4f, 0x50, 0x54,
					// OptList
					0x00, 0x00, 0x00, 0x03,
					// Length
					0x00, 0x00, 0x00, 0x00,
				}},
				// nbdOptReply{REPLYMAGIC, OptList, RepServer, 7}
				{opRead, []byte{
					// REPLYMAGIC
					0x00, 0x03, 0xe8, 0x89, 0x04, 0x55, 0x65, 0xa9,
					// OptList
					0x00, 0x00, 0x00, 0x03,
					// RepServer
					0x00, 0x00, 0x00, 0x02,
					// Length
					0x00, 0x00, 0x00, 0x07,
				}},
				// uint32(3)
				{opRead, []byte{0x00, 0x00, 0x00, 0x03}},
				// "foo"
				{opRead, []byte{0x66, 0x6f, 0x6f}},
				// nbdOptReply{REPLYMAGIC, OptList, RepAck, 0}
				{opRead, []byte{
					// REPLYMAGIC
					0x00, 0x03, 0xe8, 0x89, 0x04, 0x55, 0x65, 0xa9,
					// OptList
					0x00, 0x00, 0x00, 0x03,
					// RepAck
					0x00, 0x00, 0x00, 0x01,
					// Length
					0x00, 0x00, 0x00, 0x00,
				}},
				{opClose, nil},
			},
			expectConnClosed: false,
			expectErr:        true, // EOF
		},
		{
			desc: "Request export with a name too long",
			ops: []operation{
				{opRead, fixedNewStyleHeader},
				{opWrite, fixedNewStyleClientHeader},
				// nbdClientOptions{OptMagic: IHAVEOPT, Option: OptExportName, Length: maxSafeOptionLength + 1}
				{opWrite, []byte{
					// IHAVEOPT
					0x49, 0x48, 0x41, 0x56, 0x45, 0x4f, 0x50, 0x54,
					// OptExportName
					0x00, 0x00, 0x00, 0x01,
					// maxSafeOptionLength + 1 = 4097
					0x00, 0x00, 0x10, 0x01,
				}},
			},
			expectConnClosed: true,
			expectErr:        true,
		},
		{
			desc: "Unknown export",
			ops: []operation{
				{opRead, fixedNewStyleHeader},
				{opWrite, fixedNewStyleClientHeader},
				// nbdClientOptions{OptMagic: IHAVEOPT, Option: OptExportName, Length: 3}
				{opWrite, []byte{
					// IHAVEOPT
					0x49, 0x48, 0x41, 0x56, 0x45, 0x4f, 0x50, 0x54,
					// OptExportName
					0x00, 0x00, 0x00, 0x01,
					// Length = 3
					0x00, 0x00, 0x00, 0x03,
				}},
				// "foo"
				{opWrite, []byte{0x66, 0x6f, 0x6f}},
			},
			expectConnClosed: true,
			expectErr:        true,
		},
	} {
		runTest(t, c)
	}
}
