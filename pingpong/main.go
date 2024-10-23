package main

import (
	"fmt"
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

		/*go func() {
			buf := make([]byte, 1400)
			copy(buf, "Hello Back")
			for {
				n, err := stream.Write(buf)
				if err != nil {
					log.Fatal(err)
				}
				time.Sleep(3 * time.Second)
				Log.Info("Wrote ", n, " bytes via stream")
			}
		}()*/

		buf := make([]byte, 1400)
		for {
			n, err := stream.Read(buf)
			if err != nil {
				log.Fatal(err)
			}

			fmt.Println("Received ", n, " bytes via stream")
			fmt.Println(string(buf))
		}

		// }

	} else {
		remoteAddr := os.Args[3]
		fmt.Println(remoteAddr)
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

		go func() {
			buf := make([]byte, 1400)
			for {
				n, err := stream.Read(buf)
				if err != nil {
					log.Fatal(err)
				}

				fmt.Println("Received ", n, " bytes via stream")
				fmt.Println(string(buf))
			}
		}()

		buf := make([]byte, 1400)
		// Write hello world into buf
		copy(buf, "Hello World")
		for {
			n, err := stream.Write(buf)
			if err != nil {
				log.Fatal(err)
			}

			fmt.Println("Sent ", n, " bytes via stream")
			time.Sleep(1 * time.Second)
		}

	}

}
