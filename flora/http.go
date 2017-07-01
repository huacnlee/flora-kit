package flora

import (
	"net"
	"io"
	"net/http"
	"bufio"
	"bytes"
	"net/http/httputil"
	"log"
)

// local socks server  connect
func httpProxyConnect(conn net.Conn, first byte) (addr string, hostType int, raw []byte, err error) {
	var (
		HTTP_200 = []byte("HTTP/1.1 200 Connection Established\r\n\r\n")
		host     string
		port     string
	)

	buf := make([]byte, 4096)
	buf[0] = first
	io.ReadAtLeast(conn, buf[1:], 1)
	req, err := http.ReadRequest(bufio.NewReader(bytes.NewReader(buf)))
	if nil != err {
		log.Print("cast []byte to request has error ", err)
		return
	}
	host, port, err = net.SplitHostPort(req.Host)
	if nil != err {
		host = req.Host
		port = req.URL.Port()
	}
	scheme := req.URL.Scheme
	if "" == port {
		if scheme == "http" {
			port = "80"
		} else {
			port = "443"
		}
	}
	addr = net.JoinHostPort(host, port)
	method := req.Method
	//path := req.URL.Path
	switch method {
	case http.MethodConnect:
		hostType = getRequestType(addr)
		_, err = conn.Write(HTTP_200)
	default:
		removeHeaders(req)
		raw, err = httputil.DumpRequest(req, true)
	}

	//log.Printf(string(raw))
	return
}

func getRequestType(addr string) int  {
	host,_,_ := net.SplitHostPort(addr)
	ip := net.ParseIP(host)
	if nil == ip {
		return typeIPv4
	}
	return typeDm
}



func removeHeaders(req *http.Request) {
	req.RequestURI = ""
	req.Header.Del("Accept-Encoding")
	// curl can add that, see
	// https://jdebp.eu./FGA/web-proxy-connection-header.html
	req.Header.Del("Proxy-Connection")
	req.Header.Del("Proxy-Authenticate")
	req.Header.Del("Proxy-Authorization")
	//req.Header.Del("Referer")
	// Connection, Authenticate and Authorization are single hop Header:
	// http://www.w3.org/Protocols/rfc2616/rfc2616.txt
	// 14.10 Connection
	//   The Connection general-header field allows the sender to specify
	//   options that are desired for that particular connection and MUST NOT
	//   be communicated by proxies over further connections.
	req.Header.Del("Connection")
}
