package main

// ./filetransfer client 1-13,10.13.0.71:31002 1-10,10.10.0.71:31011 filetransfer
// ./filetransfer server 1-10,10.10.0.71:31011

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	qparts "github.com/netsys-lab/qparts"
)

func main() {

	mode := os.Args[1]
	localAddr := os.Args[2]

	if mode == "server" {
		opts := &qparts.QPartsListenOpts{PathSelectionResponsibility: qparts.QPARTS_PATH_SEL_RESPONSIBILITY_SERVER}
		listener, err := qparts.ListenWithOpts(localAddr, opts)
		if err != nil {
			log.Fatal(err)
		}

		// for {
		conn, err := listener.Accept()
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println("Accepted connection")

		stream, err := conn.AcceptStream()
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println("Accepted stream")

		var size uint64
		sizeData := make([]byte, 8)
		n, err := stream.Read(sizeData)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println("Received ", n, " bytes for size")
		size = binary.BigEndian.Uint64(sizeData)
		buf := make([]byte, size)

		n, err = io.ReadFull(stream, buf)
		if err != nil {
			log.Fatal(err)
		}

		file := string(buf)

		fileInfo, err := os.Stat(file)
		if err != nil {
			log.Fatal(err)
		}

		fileSize := fileInfo.Size()
		// Read file into bytes
		buf = make([]byte, fileSize)
		filePtr, err := os.Open(file)
		if err != nil {
			log.Fatal(err)
		}

		_, err = io.ReadFull(filePtr, buf)
		if err != nil {
			log.Fatal(err)
		}

		fileSizeData := make([]byte, 8)
		binary.BigEndian.PutUint64(fileSizeData, uint64(fileSize))

		n, err = stream.Write(fileSizeData)
		if err != nil {
			log.Fatal(err)
		}

		n, err = stream.Write(buf)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("-------------------------------")
		fmt.Printf("Hash %x\n", sha256.Sum256(buf))
		fmt.Println("Sent ", n, " bytes via stream")
		time.Sleep(1 * time.Second)

		// }

	} else {
		remoteAddr := os.Args[3]
		file := os.Args[4]
		fmt.Println(remoteAddr)
		opts := &qparts.QPartsDialOpts{PathSelectionResponsibility: qparts.QPARTS_PATH_SEL_RESPONSIBILITY_SERVER}
		conn, err := qparts.DialWithOpts(localAddr, remoteAddr, opts)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println("Opened connection")

		stream, err := conn.OpenStream()
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println("Opened stream")

		size := len(file)
		sizeData := make([]byte, 8)
		binary.BigEndian.PutUint64(sizeData, uint64(size))

		n, err := stream.Write(sizeData)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("Sent ", n, " bytes for size")

		// Read file into bytes
		buf := []byte(file)
		n, err = stream.Write(buf)
		if err != nil {
			log.Fatal(err)
		}

		var fileSize uint64
		fileSizeData := make([]byte, 8)
		n, err = stream.Read(fileSizeData)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println("Received ", n, " bytes for size")
		fileSize = binary.BigEndian.Uint64(fileSizeData)
		buf = make([]byte, fileSize)

		n, err = io.ReadFull(stream, buf)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println("-------------------------------")
		fmt.Println("Received ", n, " bytes via stream")
		fmt.Printf("Hash %x\n", sha256.Sum256(buf))

	}

}
