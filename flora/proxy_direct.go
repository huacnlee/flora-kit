package flora

import "net"

type DirectServer struct {
	proxyType string
}

func NewDirect() (*DirectServer) {
	return &DirectServer{proxyType: ServerTypeDirect}
}

func (s *DirectServer) FailCount() int {
	return 0
}

func (s *DirectServer) ResetFailCount()  {

}

func (s *DirectServer) AddFail() {

}

func (s *DirectServer) ProxyType() string {
	return s.proxyType
}

func (s *DirectServer) DialWithRawAddr(raw []byte, host string) (remote net.Conn, err error) {
	conn, err := net.Dial("tcp", host)
	if nil != err{
		return nil,err
	}
	if  nil != raw && len(raw) > 0 {
		conn.Write(raw)
	}
	return conn, err
}
