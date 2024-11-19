package qpcrypto

import (
	"crypto/tls"
	"log"

	"github.com/netsys-lab/qparts/pkg/qpenv"
	scionpila "github.com/netsys-lab/scion-pila"
	"github.com/scionproto/scion/pkg/snet"
)

func NewQPartsListenTLSConfig(local *snet.UDPAddr) (*tls.Config, error) {

	if !qpenv.PilaSettings.Enabled {
		certs := MustGenerateSelfSignedCert()
		return &tls.Config{InsecureSkipVerify: true, Certificates: certs, NextProtos: []string{"qparts"}}, nil
	}

	client := scionpila.NewSCIONPilaClient(qpenv.PilaSettings.ServerAddr)

	key := scionpila.NewPrivateKey()
	csr, err := scionpila.NewCertificateSigningRequest(key)
	if err != nil {
		log.Fatal(err)
	}

	certificate, err := client.FetchCertificateFromSigningRequest(local.String(), csr)
	if err != nil {
		log.Fatal(err)
	}

	tlsCerts, err := scionpila.CreateTLSCertificate(certificate, key)
	if err != nil {
		log.Fatal(err)
	}

	return &tls.Config{InsecureSkipVerify: true, Certificates: tlsCerts, NextProtos: []string{"qparts"}}, nil
}

func NewQPartsDialTLSConfig(remote *snet.UDPAddr) *tls.Config {

	if !qpenv.PilaSettings.Enabled {
		return &tls.Config{InsecureSkipVerify: true, NextProtos: []string{"qparts"}}
	}

	remoteVerifyFunc := scionpila.VerifyQUICCertificateChainsHandler(qpenv.PilaSettings.TrcFolder, remote.String())
	return &tls.Config{InsecureSkipVerify: true, VerifyPeerCertificate: remoteVerifyFunc, NextProtos: []string{"qparts"}}
}
