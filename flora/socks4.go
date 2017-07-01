package flora

import (
	"io"
	"net"
	"encoding/binary"
	"fmt"
)

/*
socks4 protocol

request
byte | 0  | 1 | 2 | 3 | 4 | 5 | 6 | 7 | 8 | ...  |
     |0x04|cmd| port  |     ip        |  user\0  |

reply
byte | 0  |  1   | 2 | 3 | 4 | 5 | 6 | 7|
     |0x00|status|       |              |


socks4a protocol

request
byte | 0  | 1 | 2 | 3 |4 | 5 | 6 | 7 | 8 | ... |...     |
     |0x04|cmd| port  |  0.0.0.x     |  user\0 |domain\0|

reply
byte | 0  |  1  | 2 | 3 | 4 | 5 | 6| 7 |
	 |0x00|staus| port  |    ip        |

*/

// local socks server  connect
func socks4Connect(conn net.Conn,first byte ) (addr string, hostType int, err error) {
	const (
		idVer     = 0
		idStatus  = 1
		idPort    = 2 // address type index
		idPortLen = 2
		idIP      = 4 // ip addres start index
		idIPLen   = 4 // domain address length index

		idVariable = 8
		id4aFixLen = 8
		cmdConnect = 1
	)
	// refer to getRequest in flora.go for why set buffer size to 263
	buf := make([]byte, 128)
	buf[idVer] = first
	var n int
	// read till we get possible domain length field
	if n, err = io.ReadAtLeast(conn, buf[1:], id4aFixLen); err != nil {
		return
	}
	n ++
	// command only support connect
	if buf[idStatus] != cmdConnect {
		return
	}
	// get port
	port := binary.BigEndian.Uint16(buf[idPort:idPort+idPortLen])

	// get ip
	ip := net.IP(buf[idIP:idIP+idIPLen])
	hostType = typeIPv4
	var host = ip.String()

	//socks4a
	if ip[0] == 0x00 && ip[1] == 0x00 && ip[2] == 0x00 && ip[3] != 0x00 && n+1 >= id4aFixLen {
		dm := buf[idVariable:n]
		host = string(dm)
		hostType = typeDm
	}
	addr = net.JoinHostPort(host, fmt.Sprintf("%d", port))
	_, err = conn.Write([]byte{0x00, 0x5a, 0x01, 0x02, 0x00, 0x00, 0x00, 0x00})
	return
}
