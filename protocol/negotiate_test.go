package protocol

import (
	"bytes"
	"encoding/binary"
	"io"
	"net"
	"sync"
	"testing"

	"github.com/anupcshan/gonbd/blockdevice"
)

func expectRead(t *testing.T, c net.Conn, s interface{}) {
	t.Helper()
	expected := new(bytes.Buffer)
	if err := binary.Write(expected, binary.BigEndian, s); err != nil {
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

func expectWrite(t *testing.T, c net.Conn, s interface{}) {
	t.Helper()
	if err := binary.Write(c, binary.BigEndian, s); err != nil {
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
	data interface{}
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
	sc := new(serverConnection)
	sc.conn = serverConn
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

	err := sc.negotiate()
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

func TestNegotiation(t *testing.T) {
	for _, c := range []testcase{
		{
			desc: "Handle abort",
			ops: []operation{
				{opRead, nbdFixedNewStyleHeader{NBDMAGIC, IHAVEOPT, FlagFixedNewStyle | FlagNoZeroes}},
				{opWrite, nbdClientFlags(0)},
				{opWrite, nbdClientOptions{OptMagic: IHAVEOPT, Option: OptAbort}},
				{opRead, nbdOptReply{REPLYMAGIC, OptAbort, RepAck, 0}},
			},
			expectConnClosed: true,
			expectErr:        true,
		},
		{
			desc: "Handle list with 0 exports",
			ops: []operation{
				{opRead, nbdFixedNewStyleHeader{NBDMAGIC, IHAVEOPT, FlagFixedNewStyle | FlagNoZeroes}},
				{opWrite, nbdClientFlags(0)},
				{opWrite, nbdClientOptions{OptMagic: IHAVEOPT, Option: OptList}},
				{opRead, nbdOptReply{REPLYMAGIC, OptList, RepAck, 0}},
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
				{opRead, nbdFixedNewStyleHeader{NBDMAGIC, IHAVEOPT, FlagFixedNewStyle | FlagNoZeroes}},
				{opWrite, nbdClientFlags(0)},
				{opWrite, nbdClientOptions{OptMagic: IHAVEOPT, Option: OptList}},
				{opRead, nbdOptReply{REPLYMAGIC, OptList, RepServer, 7}},
				{opRead, uint32(3)},
				{opRead, []byte("foo")},
				{opRead, nbdOptReply{REPLYMAGIC, OptList, RepAck, 0}},
				{opClose, nil},
			},
			expectConnClosed: false,
			expectErr:        true, // EOF
		},
		{
			desc: "Request export with a name too long",
			ops: []operation{
				{opRead, nbdFixedNewStyleHeader{NBDMAGIC, IHAVEOPT, FlagFixedNewStyle | FlagNoZeroes}},
				{opWrite, nbdClientFlags(0)},
				{opWrite, nbdClientOptions{OptMagic: IHAVEOPT, Option: OptExportName, Length: maxSafeOptionLength + 1}},
			},
			expectConnClosed: true,
			expectErr:        true,
		},
		{
			desc: "Unknown export",
			ops: []operation{
				{opRead, nbdFixedNewStyleHeader{NBDMAGIC, IHAVEOPT, FlagFixedNewStyle | FlagNoZeroes}},
				{opWrite, nbdClientFlags(0)},
				{opWrite, nbdClientOptions{OptMagic: IHAVEOPT, Option: OptExportName, Length: 3}},
				{opWrite, []byte("foo")},
			},
			expectConnClosed: true,
			expectErr:        true,
		},
	} {
		runTest(t, c)
	}
}
