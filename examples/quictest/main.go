package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/binary"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	mrand "math/rand"
	"net"
	"time"

	optimizedconn "github.com/netsys-lab/scion-optimized-connection/pkg"
	"github.com/quic-go/quic-go"
	"github.com/scionproto/scion/pkg/snet"
)

// MustGenerateSelfSignedCert generates private key and a self-signed dummy
// certificate usable for TLS with InsecureSkipVerify: true.
// Like GenerateSelfSignedCert but panics on error and returns a slice with a
// single entry, for convenience when initializing a tls.Config structure.
func MustGenerateSelfSignedCert() []tls.Certificate {
	cert, err := GenerateSelfSignedCert()
	if err != nil {
		panic(err)
	}
	return []tls.Certificate{*cert}
}

// GenerateSelfSignedCert generates a private key and a self-signed dummy
// certificate usable for TLS with InsecureSkipVerify: true
func GenerateSelfSignedCert() (*tls.Certificate, error) {
	priv, err := rsaGenerateKey()
	if err != nil {
		return nil, err
	}
	return createCertificate(priv)
}

func rsaGenerateKey() (*rsa.PrivateKey, error) {
	return rsa.GenerateKey(rand.Reader, 2048)
}

// createCertificate creates a self-signed dummy certificate for the given key
// Inspired/copy pasted from crypto/tls/generate_cert.go
func createCertificate(priv *rsa.PrivateKey) (*tls.Certificate, error) {
	notBefore := time.Now()
	notAfter := notBefore.Add(365 * 24 * time.Hour)

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, fmt.Errorf("failed to generate serial number: %w", err)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"scionlab"},
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{"dummy"},
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return nil, err
	}

	certPEMBuf := &bytes.Buffer{}
	if err := pem.Encode(certPEMBuf, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}); err != nil {
		return nil, err
	}

	privBytes, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return nil, fmt.Errorf("unable to marshal private key: %w", err)
	}

	keyPEMBuf := &bytes.Buffer{}
	if err := pem.Encode(keyPEMBuf, &pem.Block{Type: "PRIVATE KEY", Bytes: privBytes}); err != nil {
		return nil, err
	}

	cert, err := tls.X509KeyPair(certPEMBuf.Bytes(), keyPEMBuf.Bytes())
	return &cert, err
}

type connectedPacketConn struct {
	net.Conn
}

func (c connectedPacketConn) WriteTo(b []byte, to net.Addr) (int, error) {
	return c.Write(b)
}

func (c connectedPacketConn) ReadFrom(b []byte) (int, net.Addr, error) {
	n, err := c.Read(b)
	return n, c.RemoteAddr(), err
}

type PartsPacketPacker struct {
}

func NewPartsPacketPacker() *PartsPacketPacker {
	return &PartsPacketPacker{}
}

type PartsDataPacket struct {
	Flags          uint32
	StreamId       uint64
	PartId         uint64
	PartSize       uint32
	PathId         uint32
	SequenceNumber uint64
	Data           []byte
}

/*
 * Data Packet Handling:
 * 	- Flags: 4 bytes
 * 	- Stream ID: 8 bytes
 * 	- Sequence Number: 8 bytes
 * 	- Path ID: 4 bytes
 *  - Part ID: 8 bytes
 *  - PartSize: 4 bytes
 * 	- Data: variable length
 */
func (bp *PartsPacketPacker) GetDataHeaderLen() int {
	return 36
}

func (bp *PartsPacketPacker) PackData(buf *[]byte, packet *PartsDataPacket) error {
	// Log.Info("Bufferlen ", len(*buf))
	// TODO: Maybe set flags here
	binary.BigEndian.PutUint32((*buf)[0:4], packet.Flags)
	binary.BigEndian.PutUint64((*buf)[4:12], packet.StreamId)
	binary.BigEndian.PutUint64((*buf)[12:20], packet.SequenceNumber)
	binary.BigEndian.PutUint32((*buf)[20:24], packet.PathId)
	binary.BigEndian.PutUint64((*buf)[24:32], packet.PartId)
	binary.BigEndian.PutUint32((*buf)[32:36], packet.PartSize)
	return nil
}

func (bp *PartsPacketPacker) UnpackData(buf *[]byte) (*PartsDataPacket, error) {
	p := PartsDataPacket{}
	p.Flags = binary.BigEndian.Uint32((*buf)[0:4])
	p.StreamId = binary.BigEndian.Uint64((*buf)[4:12])
	p.SequenceNumber = binary.BigEndian.Uint64((*buf)[12:20])
	p.PathId = binary.BigEndian.Uint32((*buf)[20:24])
	p.PartId = binary.BigEndian.Uint64((*buf)[24:32])
	p.PartSize = binary.BigEndian.Uint32((*buf)[32:36])

	//*buf = (*buf)[36:]
	//p.Data = *buf
	return &p, nil
}

func main() {

	go func() {
		listen()
	}()

	time.Sleep(3 * time.Second)
	dial()
}

// Function to generate a random byte array of a given size
func generateRandomBytes(size int) ([]byte, error) {
	bytes := make([]byte, size)
	_, err := rand.Read(bytes)
	if err != nil {
		return nil, err
	}
	return bytes, nil
}

// Function to generate a random integer between min and max (inclusive)
func randomInt(min, max int) int {
	if min > max {
		panic("min should be less than or equal to max")
	}
	return mrand.Intn(max-min+1) + min
}

func dial() {
	remote := "1-150,127.0.0.1:4443"
	raddr, err := snet.ParseUDPAddr(remote)
	if err != nil {
		panic(err)
	}
	rAddrPort := raddr.Host.AddrPort()
	rudpAddr := net.UDPAddrFromAddrPort(rAddrPort)

	local := "1-150,127.0.0.1:11443"
	laddr, err := snet.ParseUDPAddr(local)
	if err != nil {
		panic(err)
	}

	lAddrPort := laddr.Host.AddrPort()
	ludpAddr := net.UDPAddrFromAddrPort(lAddrPort)

	h := host()
	paths, err := h.queryPaths(context.Background(), raddr.IA)
	fmt.Println(paths)

	raddr.Path = paths[0].Dataplane()

	conn, err := optimizedconn.Dial(ludpAddr, raddr)
	if err != nil {
		panic(err)
	}

	pconn := connectedPacketConn{conn}

	session, err := quic.Dial(context.Background(), pconn, rudpAddr, &tls.Config{InsecureSkipVerify: true, NextProtos: []string{"qparts"}}, &quic.Config{})
	if err != nil {
		panic(err)
	}

	stream, err := session.OpenStreamSync(context.Background())
	if err != nil {
		panic(err)
	}

	buf := make([]byte, 36)
	partsPacketPacker := NewPartsPacketPacker()

	for {
		size := randomInt(100000, 7000000)
		data, err := generateRandomBytes(size)
		if err != nil {
			panic(err)
		}
		fmt.Printf("Sending %x\n", sha256.Sum256(data))
		partsPacket := PartsDataPacket{
			Flags:          0x1,
			StreamId:       1,
			SequenceNumber: 1,
			PathId:         1,
			PartId:         1,
			PartSize:       uint32(len(data)),
		}
		err = partsPacketPacker.PackData(&buf, &partsPacket)
		if err != nil {
			panic(err)
		}
		_, err = stream.Write(buf)
		if err != nil {
			panic(err)
		}
		n, err := io.CopyN(stream, bytes.NewReader(data), int64(len(data)))
		fmt.Println("Written ", n, " of ", len(data))
		// _, err = stream.Write(data)
		if err != nil {
			panic(err)
		}
		time.Sleep(1 * time.Second)
	}

}

func listen() {
	local := "1-150,127.0.0.1:4443"
	addr, err := snet.ParseUDPAddr(local)
	if err != nil {
		panic(err)
	}

	lAddr := addr.Host.AddrPort()
	udpAddr := net.UDPAddrFromAddrPort(lAddr)
	conn, err := optimizedconn.Listen(udpAddr)
	if err != nil {
		panic(err)
	}

	pconn := connectedPacketConn{conn}

	listener, err := quic.Listen(pconn, &tls.Config{InsecureSkipVerify: true, Certificates: MustGenerateSelfSignedCert(), NextProtos: []string{"qparts"}}, &quic.Config{})
	if err != nil {
		panic(err)
	}

	for {
		sess, err := listener.Accept(context.Background())
		if err != nil {
			panic(err)
		}

		go func() {
			for {
				stream, err := sess.AcceptStream(context.Background())
				if err != nil {
					panic(err)
				}
				partsPacketPacker := NewPartsPacketPacker()
				data := make([]byte, 10000024)
				buf := make([]byte, 36)

				go func() {
					for {

						n, err := stream.Read(buf)
						if err != nil {
							panic(err)
						}
						fmt.Println("Read Header", n)

						partsPacket, err := partsPacketPacker.UnpackData(&buf)
						if err != nil {
							panic(err)
						}
						n, err = io.ReadFull(stream, data[:partsPacket.PartSize])
						if err != nil {
							panic(err)
						}
						fmt.Println("Read Data", n)
						/*n, err = stream.Read(buf)
						if err != nil {
							panic(err)
						}*/
						fmt.Printf("Received %x\n", sha256.Sum256(data[:n]))
						fmt.Println("---------------------------------")
						// fmt.Println(string())
					}

				}()
			}
		}()
	}
}
