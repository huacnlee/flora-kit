package flora

import "net"

type Reject struct {
	proxyType string
}

func NewReject() *Reject {
	return &Reject{proxyType: ServerTypeReject}
}

func (s *Reject) FailCount() int {
	return 0
}

func (s *Reject) ResetFailCount() {

}

func (s *Reject) AddFail() {

}

func (s *Reject) ProxyType() string {
	return s.proxyType
}

func (s *Reject) DialWithRawAddr(raw []byte, host string) (remote net.Conn, err error) {
	return nil, errReject
}
