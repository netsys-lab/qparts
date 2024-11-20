package main

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
		listener, err := qparts.Listen(localAddr)
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
		fmt.Println("Size: ", size)

		n, err = io.ReadFull(stream, buf)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println(" -------------------------------------- ")
		fmt.Printf("Hash %x\n", sha256.Sum256(buf))
		fmt.Println("Received ", n, " bytes via stream")

		// }

	} else {
		remoteAddr := os.Args[3]
		file := os.Args[4]
		// fmt.Println(remoteAddr)
		conn, err := qparts.Dial(localAddr, remoteAddr)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println("Opened connection")

		stream, err := conn.OpenStream()
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println("Opened stream")

		fileInfo, err := os.Stat(file)
		if err != nil {
			log.Fatal(err)
		}

		size := fileInfo.Size()
		sizeData := make([]byte, 8)
		binary.BigEndian.PutUint64(sizeData, uint64(size))

		n, err := stream.Write(sizeData)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("Sent ", n, " bytes for size")

		// Read file into bytes
		buf := make([]byte, size)
		filePtr, err := os.Open(file)
		if err != nil {
			log.Fatal(err)
		}

		_, err = io.ReadFull(filePtr, buf)
		if err != nil {
			log.Fatal(err)
		}

		n, err = stream.Write(buf)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println(" -------------------------------------- ")
		fmt.Printf("Hash %x\n", sha256.Sum256(buf))
		fmt.Println("Sent ", n, " bytes via stream")
		time.Sleep(10 * time.Second)
		// time.Sleep(1 * time.Second)

	}

}
