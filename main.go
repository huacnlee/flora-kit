package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"flora-kit/flora"
	ss "github.com/shadowsocks/shadowsocks-go/shadowsocks"
	"io"
	"log"
	"math/rand"
	"net"
	"strconv"
	"time"
	"flag"
	"strings"
)

var debug ss.DebugLog

var (
	errAddrType      = errors.New("socks addr type not supported")
	errVer           = errors.New("socks version not supported")
	errAuthExtraData = errors.New("socks authentication get extra data")
	errReqExtraData  = errors.New("socks request get extra data")
	errCmd           = errors.New("socks command not supported")
	errReject        = errors.New("socks reject this request")
	errSupported     = errors.New("proxy type not supported")
)

const (
	socksVer5       = 5
	socksCmdConnect = 1
)

func init() {
	rand.Seed(time.Now().Unix())
}

func handShake(conn net.Conn) (err error) {
	const (
		idVer     = 0
		idNmethod = 1
	)
	// version identification and method selection message in theory can have
	// at most 256 methods, plus version and nmethod field in total 258 bytes
	// the current rfc defines only 3 authentication methods (plus 2 reserved),
	// so it won't be such long in practice

	buf := make([]byte, 258)

	var n int
	ss.SetReadTimeout(conn)
	// make sure we get the nmethod field
	if n, err = io.ReadAtLeast(conn, buf, idNmethod+1); err != nil {
		return
	}
	if buf[idVer] != socksVer5 {
		return errVer
	}
	nmethod := int(buf[idNmethod])
	msgLen := nmethod + 2
	if n == msgLen { // handshake done, common case
		// do nothing, jump directly to send confirmation
	} else if n < msgLen { // has more methods to read, rare case
		if _, err = io.ReadFull(conn, buf[n:msgLen]); err != nil {
			return
		}
	} else { // error, should not get extra data
		return errAuthExtraData
	}
	// send confirmation: version 5, no authentication required
	_, err = conn.Write([]byte{socksVer5, 0})
	return
}

func getRequest(conn net.Conn) (rawaddr []byte, host string, err error) {
	const (
		idVer   = 0
		idCmd   = 1
		idType  = 3 // address type index
		idIP0   = 4 // ip addres start index
		idDmLen = 4 // domain address length index
		idDm0   = 5 // domain address start index

		typeIPv4 = 1 // type is ipv4 address
		typeDm   = 3 // type is domain address
		typeIPv6 = 4 // type is ipv6 address

		lenIPv4   = 3 + 1 + net.IPv4len + 2 // 3(ver+cmd+rsv) + 1addrType + ipv4 + 2port
		lenIPv6   = 3 + 1 + net.IPv6len + 2 // 3(ver+cmd+rsv) + 1addrType + ipv6 + 2port
		lenDmBase = 3 + 1 + 1 + 2           // 3 + 1addrType + 1addrLen + 2port, plus addrLen
	)
	// refer to getRequest in server.go for why set buffer size to 263
	buf := make([]byte, 263)
	var n int
	ss.SetReadTimeout(conn)
	// read till we get possible domain length field
	if n, err = io.ReadAtLeast(conn, buf, idDmLen+1); err != nil {
		return
	}
	// check version and cmd
	if buf[idVer] != socksVer5 {
		err = errVer
		return
	}
	if buf[idCmd] != socksCmdConnect {
		err = errCmd
		return
	}

	reqLen := -1
	switch buf[idType] {
	case typeIPv4:
		reqLen = lenIPv4
	case typeIPv6:
		reqLen = lenIPv6
	case typeDm:
		reqLen = int(buf[idDmLen]) + lenDmBase
	default:
		err = errAddrType
		return
	}

	if n == reqLen {
		// common case, do nothing
	} else if n < reqLen { // rare case
		if _, err = io.ReadFull(conn, buf[n:reqLen]); err != nil {
			return
		}
	} else {
		err = errReqExtraData
		return
	}

	rawaddr = buf[idType:reqLen]

	switch buf[idType] {
	case typeIPv4:
		host = net.IP(buf[idIP0: idIP0+net.IPv4len]).String()
	case typeIPv6:
		host = net.IP(buf[idIP0: idIP0+net.IPv6len]).String()
	case typeDm:
		host = string(buf[idDm0: idDm0+buf[idDmLen]])
	}
	port := binary.BigEndian.Uint16(buf[reqLen-2: reqLen])
	host = net.JoinHostPort(host, strconv.Itoa(int(port)))

	return
}

func connectToServer(action string, rawaddr []byte, addr string) (remote net.Conn, err error) {
	se := flora.ProxyServers.GetCipher(action)
	proxyType := strings.ToLower(se.ProxyType)
	if proxyType == "custom" || proxyType == "shadowsocks" {
		remote, err = ss.DialWithRawAddr(rawaddr, se.Server, se.ShadowSocksCipher.Copy())
		if err != nil {
			log.Println("error connecting to shadowsocks server:", err)
			if flora.ProxyServers.FailCipher(action) == -1 {
				return nil, err
			}
		}
		debug.Printf("connected to %s via %s\n", addr, se.Server)
		return
	} else if proxyType == "direct" {
		return net.Dial("tcp", addr)
	}
	return nil, errSupported
}

// Connection to the server in the order specified in the config. On
// connection failure, try the next server. A failed server will be tried with
// some probability according to its fail count, so we can discover recovered
// servers.
func createServerConn(rule *flora.HostRule, rawaddr []byte, addr string) (remote net.Conn, err error) {
	switch rule.Action {
	case flora.RULE_DIRECT:
		return net.Dial("tcp", addr)
	case flora.RULE_REJECT:
		return nil, errReject
	default:
		return connectToServer(rule.Action, rawaddr, addr)
	}
}

// 连接请求入口
func handleConnection(conn net.Conn) {
	closed := false
	defer func() {
		if !closed {
			conn.Close()
		}
	}()
	var err error = nil
	if err = handShake(conn); err != nil {
		log.Println("socks handshake:", err)
		return
	}
	rawaddr, host, err := getRequest(conn)
	if err != nil {
		log.Println("error getting request:", err)
		return
	}
	// Sending connection established message immediately to client.
	// This some round trip time for creating socks connection with the client.
	// But if connection failed, the client will get connection reset error.
	_, err = conn.Write([]byte{0x05, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x08, 0x43})
	if err != nil {
		debug.Println("send connection confirmation:", err)
		return
	}
	rule := flora.RuleOfHost(host)
	log.Printf("[%s]➔[%s]√[%s]", rule.Match, rule.Action, host)
	var remote net.Conn
	remote, err = createServerConn(rule, rawaddr, host)
	if err != nil {
		if len(flora.ProxyServers.SrvCipherGroup) > 1 {
			log.Printf("[%s]➔[%s]×[%s] Failed connect to all avaiable shadowsocks server ", rule.Match, rule.Action, host)
		}
		return
	}

	defer func() {
		if !closed {
			remote.Close()
		}
	}()

	go ss.PipeThenClose(conn, remote)
	ss.PipeThenClose(remote, conn)
	closed = true
}

func run(listenAddr string) {
	flora.ResetAllProxys()

	ln, err := net.Listen("tcp", listenAddr)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Listen socks", listenAddr)

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println("accept:", err)
			continue
		}
		go handleConnection(conn)
	}
}

func main() {
	var configFile, geoipdb string
	flag.StringVar(&configFile, "s", "flora.default.conf", "specify surge config file")
	flag.StringVar(&geoipdb, "d", "geoip.mmdb", "specify geoip db file")
	flora.LoadConfig(configFile, geoipdb)
	log.Println("Floar", flora.VERSION)
	if flora.ProxyServers.LocalSocksPort > 0 {
		run("0.0.0.0:" + fmt.Sprintf("%d", flora.ProxyServers.LocalSocksPort))
	}

}
