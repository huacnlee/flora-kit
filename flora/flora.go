package flora

import (
	ss "github.com/shadowsocks/shadowsocks-go/shadowsocks"
	"net"
	"errors"
	"log"
	"fmt"
	"strings"
	"regexp"
	"io"
)

const (
	ServerTypeShadowSocks = "shadowsocks"
	ServerTypeCustom      = "custom"
	ServerTypeHttp        = "http"
	ServerTypeHttps       = "https"
	ServerTypeDirect      = "direct"
	ServerTypeReject      = "direct"

	LocalServerSocksV5 = "localSocksv5"
	LocalServerHttp    = "localHttp"

	socksVer5       = 5
	socksVer4       = 4
	httpProxy       = 71
	socksCmdConnect = 1

	typeIPv4 = 1 // type is ipv4 address
	typeDm   = 3 // type is domain address
	typeIPv6 = 4 // type is ipv6 address
)

type ProxyServer interface {
	//proxy type
	ProxyType() string
	//dial
	DialWithRawAddr(raw []byte, host string) (remote net.Conn, err error)
	//
	FailCount() int

	AddFail()
	//
	ResetFailCount()
}

type Rule struct {
	Match  string
	Action string
}

var (
	errAddrType      = errors.New("socks addr type not supported")
	errVer           = errors.New("socks version not supported")
	errAuthExtraData = errors.New("socks authentication get extra data")
	errReqExtraData  = errors.New("socks request get extra data")
	errCmd           = errors.New("socks command not supported")
	errReject        = errors.New("socks reject this request")
	errSupported     = errors.New("proxy type not supported")
)

var proxyConfig *ProxyConfig

func Run(surgeCfg, geoipCfg string) {
	proxyConfig = LoadConfig(surgeCfg, geoipCfg)

	listenAddr := fmt.Sprintf("%s:%d", proxyConfig.LocalHost, proxyConfig.LocalSocksPort)
	ResetAllProxys()
	ln, err := net.Listen("tcp", listenAddr)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Listen socket", listenAddr)
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println("accept:", err)
			continue
		}
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	isClose := false
	defer func() {
		if !isClose {
			conn.Close()
		}
	}()
	var (
		host     string
		hostType int
		err      error
		rawData  []byte
	)

	buf := make([]byte, 1)
	io.ReadFull(conn, buf)

	first := buf[0]
	switch first {
	case socksVer5:
		err = handshake(conn, first)
		host, hostType, err = socks5Connect(conn)
	case socksVer4:
		host, hostType, err = socks4Connect(conn, first)
	default:
		host, hostType, rawData, err = httpProxyConnect(conn, first)
	}

	if nil != err {
		return
	}
	remote, err := matchRuleAndCreateConn(conn, host, hostType, rawData)
	if nil != err {
		return
	}
	//create remote connect
	defer func() {
		if !isClose {
			remote.Close()
		}
	}()
	go ss.PipeThenClose(conn, remote)
	ss.PipeThenClose(remote, conn)
	isClose = true
}

func matchRuleAndCreateConn(conn net.Conn, addr string, hostType int, raw []byte) (net.Conn, error) {
	if nil == conn{
		return nil,errors.New("...")
	}
	host, _, _ := net.SplitHostPort(addr)
	var rule *Rule
	rule = matchBypass(host)
	if nil == rule {
		switch hostType {
		case typeIPv4, typeIPv6:
			rule = matchIpRule(host)
		case typeDm:
			rule = matchDomainRule(host)
		}
	}
	if nil == rule {
		if nil != proxyConfig.ruleFinal {
			rule = proxyConfig.ruleFinal
		} else {
			rule = &Rule{Match: "default", Action: ServerTypeDirect}
		}
	}
	return createRemoteConn(raw, rule, addr)
}

func matchDomainRule(domain string) (*Rule) {
	for _, rule := range proxyConfig.ruleSuffixDomains {
		if strings.HasSuffix(domain, rule.Match) {
			return rule
		}
	}
	for _, rule := range proxyConfig.rulePrefixDomains {
		if strings.HasPrefix(domain, rule.Match) {
			return rule
		}
	}
	for _, rule := range proxyConfig.ruleKeywordDomains {
		if strings.Contains(domain, rule.Match) {
			return rule
		}
	}
	return nil
}

func matchIpRule(addr string) (*Rule) {
	ips := resolveRequestIPAddr(addr)
	if nil != ips {
		country := strings.ToLower(GeoIPs(ips))
		for _, rule := range proxyConfig.ruleGeoIP {
			if len(country) != 0 && strings.ToLower(rule.Match) == country {
				return rule
			}
		}
	}
	return nil
}

func matchBypass(addr string) (*Rule) {
	ip := net.ParseIP(addr)
	for _, h := range proxyConfig.bypassDomains {
		var bypass bool = false
		var isIp = nil != ip
		switch h.(type) {
		case net.IP:
			if isIp {
				bypass = ip.Equal(h.(net.IP))
			}
		case *net.IPNet:
			if isIp {
				bypass = h.(*net.IPNet).Contains(ip)
			}
		case string:
			dm := h.(string)
			r, err := regexp.Compile(dm)
			if err != nil {
				continue
			}
			bypass = r.MatchString(addr)
		}
		if bypass {
			return &Rule{Match: "bypass", Action: ServerTypeDirect}
		}
	}
	return nil
}

func createRemoteConn(raw []byte, rule *Rule, host string) (net.Conn, error) {
	server := proxyConfig.GetProxyServer(rule.Action)
	conn, err := server.DialWithRawAddr(raw, host)
	if nil != err {
		log.Printf("[%s]->[%s] 💊 ️[%s]", rule.Match, rule.Action, host)
		server.AddFail()
	} else {
		log.Printf("[%s]->[%s] ✅ [%s]", rule.Match, rule.Action, host)
		server.ResetFailCount()
	}
	return conn, err
}
