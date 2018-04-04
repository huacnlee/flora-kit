package flora

import (
	"bytes"
	"log"
	"os/exec"
	"strings"
	"net"
)

type darwin struct {
	bypassDomains []string
	address       string
}

type execNetworkFunc func(name string)

var allow_services = "Wi-Fi|Thunderbolt Bridge|Thunderbolt Ethernet"

func (d *darwin) TurnOffGlobProxy() {
	execNetworks(func(name string) {
		runNetworksetup("-setftpproxystate", name, "off")
		runNetworksetup("-setwebproxystate", name, "off")
		runNetworksetup("-setsecurewebproxystate", name, "off")
		runNetworksetup("-setstreamingproxystate", name, "off")
		runNetworksetup("-setgopherproxystate", name, "off")
		runNetworksetup("-setsocksfirewallproxystate", name, "off")
		runNetworksetup("-setproxyautodiscovery", name, "off")
	})
}

func (d *darwin) TurnOnGlobProxy() {
	host, port, _ := net.SplitHostPort(d.address)

	execNetworks(func(name string) {
		runNetworksetup("-setsocksfirewallproxy", name, host, port)
	})

	execNetworks(func(name string) {
		args := []string{"-setproxybypassdomains", name}
		args = append(args, d.bypassDomains...)
		runNetworksetup(args...)
	})
}

func runNetworksetup(args ...string) string {

	// log.Println("networksetup", args)
	cmd := exec.Command("networksetup", args...)
	var out, stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		log.Println(err)
		log.Println(stderr.String())
	}
	return out.String()
}

func execNetworks(callback execNetworkFunc) {
	for _, name := range listNetworks() {
		if !strings.Contains(allow_services, name) {
			continue
		}
		callback(name)
	}
}

func listNetworks() (networks []string) {
	out := runNetworksetup("-listallnetworkservices")
	out = strings.TrimSpace(out)
	networks = strings.Split(out, "\n")
	return
}
