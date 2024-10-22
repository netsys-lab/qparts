package qparts

import "github.com/scionproto/scion/pkg/snet"

func Listen(localAddr string) (*QPartsListener, error) {

	addr, err := snet.ParseUDPAddr(localAddr)
	if err != nil {
		return nil, err
	}

	pc := NewQPartsListener(addr)
	return pc, nil
}

func ListenAndAccept(localAddr string) (*QPartsConn, error) {

	addr, err := snet.ParseUDPAddr(localAddr)
	if err != nil {
		return nil, err
	}

	pc := NewQPartsListener(addr)

	conn, err := pc.Accept()
	if err != nil {
		return nil, err
	}

	return conn, nil
}

func Dial(localAddr, remoteAddr string) (*QPartsConn, error) {

	lAddr, err := snet.ParseUDPAddr(localAddr)
	if err != nil {
		return nil, err
	}

	rAddr, err := snet.ParseUDPAddr(remoteAddr)
	if err != nil {
		return nil, err
	}

	pc := NewQPartsConn(lAddr)
	err = pc.DialAndOpen(rAddr)
	if err != nil {
		return nil, err
	}

	return pc, nil
}
