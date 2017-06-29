package flora

import (
	"net"
	ss "github.com/shadowsocks/shadowsocks-go/shadowsocks"
	"sync"
)

type ShadowSocksServer struct {
	proxyType string
	server    string
	cipher    *ss.Cipher
	failCount int
	lock      sync.RWMutex
}

func NewShadowSocks(server string, cipher *ss.Cipher) (*ShadowSocksServer) {
	return &ShadowSocksServer{
		proxyType: ServerTypeShadowSocks,
		server:    server,
		cipher:    cipher,
	}
}

func (s *ShadowSocksServer) ResetFailCount() {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.failCount = 0
}

func (s *ShadowSocksServer) AddFail() {
	s.failCount ++
}

func (s *ShadowSocksServer) FailCount() int {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.failCount
}

func (s *ShadowSocksServer) ProxyType() string {
	return s.proxyType
}

func (s *ShadowSocksServer) DialWithRawAddr(raw []byte, host string) (net.Conn, error) {
	if nil != raw && len(raw) > 0 {
		return ss.DialWithRawAddr(raw, s.server, s.cipher.Copy())
	} else {
		return ss.Dial(host, s.server, s.cipher.Copy())
	}
}
