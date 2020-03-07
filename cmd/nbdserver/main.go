package main

import (
	"context"
	"flag"
	"log"
	"net"
	"os"

	"github.com/anupcshan/gonbd/blockdevice"
	"github.com/anupcshan/gonbd/blockdevice/file"
	"github.com/anupcshan/gonbd/protocol"
)

func main() {
	log.SetFlags(log.Lmicroseconds | log.Lshortfile)

	listenAddr := flag.String("listen", "0.0.0.0:10809", "Address to listen on")
	filePath := flag.String("file", "/tmp/nbd-backing-file", "Path to the backing file. This file should already exist")
	exportName := flag.String("export", "default", "Name of the file-backed export")

	flag.Parse()

	ln, err := net.Listen("tcp", *listenAddr)
	if err != nil {
		log.Fatal(err)
	}

	f, err := os.OpenFile(*filePath, os.O_RDWR, 0755)
	if err != nil {
		log.Fatal(err)
	}

	srv := &protocol.NbdServer{
		Exports: map[string]blockdevice.BlockDevice{
			*exportName: file.NewFileDevice(f),
		},
	}
	srv.Serve(context.Background(), ln)
}
