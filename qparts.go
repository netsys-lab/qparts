package qparts

import "github.com/scionproto/scion/pkg/snet"

const (
	QPARTS_PATH_SEL_RESPONSIBILITY_CLIENT = 1
	QPARTS_PATH_SEL_RESPONSIBILITY_SERVER = 2
)

type QPartsDialOpts struct {
	PathSelectionResponsibility uint8
}

type QPartsListenOpts struct {
	PathSelectionResponsibility uint8
}

func Listen(localAddr string) (*QPartsListener, error) {
	return ListenWithOpts(localAddr, nil)
}

func ListenWithOpts(localAddr string, opts *QPartsListenOpts) (*QPartsListener, error) {

	addr, err := snet.ParseUDPAddr(localAddr)
	if err != nil {
		return nil, err
	}

	pc := NewQPartsListener(addr, opts)

	return pc, nil
}

func ListenAndAccept(localAddr string) (*QPartsConn, error) {
	return ListenAndAcceptWithOpts(localAddr, nil)
}

func ListenAndAcceptWithOpts(localAddr string, opts *QPartsListenOpts) (*QPartsConn, error) {

	addr, err := snet.ParseUDPAddr(localAddr)
	if err != nil {
		return nil, err
	}

	pc := NewQPartsListener(addr, opts)

	conn, err := pc.Accept()
	if err != nil {
		return nil, err
	}

	return conn, nil
}

func Dial(localAddr, remoteAddr string) (*QPartsConn, error) {
	return DialWithOpts(localAddr, remoteAddr, nil)
}

func DialWithOpts(localAddr, remoteAddr string, opts *QPartsDialOpts) (*QPartsConn, error) {

	lAddr, err := snet.ParseUDPAddr(localAddr)
	if err != nil {
		return nil, err
	}

	rAddr, err := snet.ParseUDPAddr(remoteAddr)
	if err != nil {
		return nil, err
	}

	pc := NewQPartsConn(lAddr)
	err = pc.DialAndOpen(rAddr, opts)
	if err != nil {
		return nil, err
	}

	return pc, nil
}
