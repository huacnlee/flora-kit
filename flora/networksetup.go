package flora

import (
	"bytes"
	"fmt"
	"log"
	"os/exec"
	"runtime"
	"strings"
)

type execNetworkFunc func(name string)

var allow_services = "Wi-Fi|Thunderbolt Bridge|Thunderbolt Ethernet"

func ResetAllProxys() {
	if runtime.GOOS != "darwin" {
		log.Println("WARN: Your not in macOS, Networksetup skiped. Please change Network proxy setting by manually.")
	}

	execNetworks(func(name string) {
		runNetworksetup("-setftpproxystate", name, "off")
		runNetworksetup("-setwebproxystate", name, "off")
		runNetworksetup("-setsecurewebproxystate", name, "off")
		runNetworksetup("-setstreamingproxystate", name, "off")
		runNetworksetup("-setgopherproxystate", name, "off")
		runNetworksetup("-setsocksfirewallproxystate", name, "on")
		runNetworksetup("-setproxyautodiscovery", name, "off")
	})
}

func SetSocksFirewallProxy() {
	execNetworks(func(name string) {
		runNetworksetup("-setsocksfirewallproxy", name, "127.0.0.1", fmt.Sprintf("%d", DEFAULT_SOCKS_PORT))
	})
}

func SetProxyBypassDomains(domains []string) {
	execNetworks(func(name string) {
		args := []string{"-setproxybypassdomains", name}
		args = append(args, domains...)
		runNetworksetup(args...)
	})
}

func runNetworksetup(args ...string) string {
	if runtime.GOOS != "darwin" {
		return ""
	}

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
