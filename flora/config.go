package flora

import (
	"fmt"
	"github.com/go-ini/ini"
	ss "github.com/shadowsocks/shadowsocks-go/shadowsocks"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
)

const (
	SOCKS_PORT  = 7657
	RULE_REJECT = 0
	RULE_DIRECT = 1
	RULE_PROXY  = 2
)

type DomainRule struct {
	S string
	T int
}

var ruleSuffixDomains = []*DomainRule{}
var rulePrefixDomains = []*DomainRule{}
var ruleKeywordDomains = []*DomainRule{}
var ruleGeoIP = &DomainRule{}

var ssConfig ss.Config
var iniConfig *ini.File

type ProxyServerCipher struct {
	Server string
	Cipher *ss.Cipher
}

var ProxyServers struct {
	SrvCipher []*ProxyServerCipher
	FailCnt   []int // failed connection count
}

func init() {
	var configFilename = "/Users/jason/Dropbox/Surge/Default.conf"
	var userConfigFilename = "/Users/jason/Dropbox/Surge/User.conf"
	var iniOpts = ini.LoadOptions{
		AllowBooleanKeys: true,
		Loose:            true,
		Insensitive:      true,
	}
	cfg, err := ini.LoadSources(iniOpts, configFilename, userConfigFilename)

	if err != nil {
		panic(fmt.Sprintf("Config file %v not found, or have error: \n\t%v", configFilename, err))
	}
	iniConfig = cfg

	loadProxy()
	loadRules()

	log.Println(RuleOfHost("www.google.com"))
	log.Println(RuleOfHost("www.twitter.com"))
}

// [Proxy] Section
func loadProxy() {
	ssConfig.LocalPort = SOCKS_PORT
	for _, key := range iniConfig.Section("Proxy").Keys() {
		var proxys = readArrayLine(key.String())
		// ShadowSocks Proxys
		if proxys[0] == "custom" || proxys[0] == "shadowsocks" {
			var server = strings.Join(proxys[1:3], ":")
			var serverInfo = []string{server, proxys[4], proxys[3]}
			ssConfig.ServerPassword = append(ssConfig.ServerPassword, serverInfo)
		}
	}

	if ssConfig.Method == "" {
		ssConfig.Method = "aes-256-cfb"
	}
	if len(ssConfig.ServerPassword) == 0 {
		if !enoughSSOptions(&ssConfig) {
			fmt.Fprintln(os.Stderr, "must specify server address, password and both server/local port")
			os.Exit(1)
		}
	} else {
		if ssConfig.LocalPort == 0 {
			fmt.Fprintln(os.Stderr, "must specify local port")
			os.Exit(1)
		}
	}

	parseServerConfig()
}

// 载入 [Rule]
func loadRules() {
	for _, key := range iniConfig.Section("Rule").KeyStrings() {
		var items = readArrayLine(key)
		var ruleType = RULE_DIRECT
		if len(items) >= 3 {
			switch items[2] {
			case "proxy":
				ruleType = RULE_PROXY
			case "direct":
				ruleType = RULE_DIRECT
			case "reject":
				ruleType = RULE_REJECT
			default:
				ruleType = RULE_DIRECT
			}
		}

		switch items[0] {
		case "domain-suffix":
			ruleSuffixDomains = append(ruleSuffixDomains, &DomainRule{S: items[1], T: ruleType})
		case "domain-prefix":
			rulePrefixDomains = append(rulePrefixDomains, &DomainRule{S: items[1], T: ruleType})
		case "domain-keyword":
			ruleKeywordDomains = append(ruleKeywordDomains, &DomainRule{S: items[1], T: ruleType})
		case "geoip":
			ruleGeoIP = &DomainRule{S: items[1], T: ruleType}
		}
	}
}

func enoughSSOptions(config *ss.Config) bool {
	return config.Server != nil && config.ServerPort != 0 &&
		config.LocalPort != 0 && config.Password != ""
}

func readArrayLine(source string) []string {
	out := strings.Split(source, ",")
	for i, str := range out {
		out[i] = strings.TrimSpace(str)
	}
	return out
}

func RuleOfHost(host string) *DomainRule {
	hostParts := strings.Split(host, ":")
	domain := hostParts[0]

	for _, rule := range ruleSuffixDomains {
		if strings.HasSuffix(domain, rule.S) {
			return rule
		}
	}

	for _, rule := range rulePrefixDomains {
		if strings.HasPrefix(domain, rule.S) {
			return rule
		}
	}

	for _, rule := range ruleKeywordDomains {
		if strings.Contains(domain, rule.S) {
			return rule
		}
	}
	return &DomainRule{S: "", T: RULE_DIRECT}
}

func parseServerConfig() {
	config := ssConfig
	hasPort := func(s string) bool {
		_, port, err := net.SplitHostPort(s)
		if err != nil {
			return false
		}
		return port != ""
	}

	if len(config.ServerPassword) == 0 {
		method := config.Method
		if config.Auth {
			method += "-auth"
		}
		// only one encryption table
		cipher, err := ss.NewCipher(method, config.Password)
		if err != nil {
			log.Fatal("Failed generating ciphers:", err)
		}
		srvPort := strconv.Itoa(config.ServerPort)
		srvArr := config.GetServerArray()
		n := len(srvArr)
		ProxyServers.SrvCipher = make([]*ProxyServerCipher, n)

		for i, s := range srvArr {
			if hasPort(s) {
				log.Println("ignore server_port option for server", s)
				ProxyServers.SrvCipher[i] = &ProxyServerCipher{s, cipher}
			} else {
				ProxyServers.SrvCipher[i] = &ProxyServerCipher{net.JoinHostPort(s, srvPort), cipher}
			}
		}
	} else {
		// multiple servers
		n := len(config.ServerPassword)
		ProxyServers.SrvCipher = make([]*ProxyServerCipher, n)

		cipherCache := make(map[string]*ss.Cipher)
		i := 0
		for _, serverInfo := range config.ServerPassword {
			if len(serverInfo) < 2 || len(serverInfo) > 3 {
				log.Fatalf("server %v syntax error\n", serverInfo)
			}
			server := serverInfo[0]
			passwd := serverInfo[1]
			encmethod := ""
			if len(serverInfo) == 3 {
				encmethod = serverInfo[2]
			}
			if !hasPort(server) {
				log.Fatalf("no port for server %s\n", server)
			}
			// Using "|" as delimiter is safe here, since no encryption
			// method contains it in the name.
			cacheKey := encmethod + "|" + passwd
			cipher, ok := cipherCache[cacheKey]
			if !ok {
				var err error
				cipher, err = ss.NewCipher(encmethod, passwd)
				if err != nil {
					log.Fatal("Failed generating ciphers:", err)
				}
				cipherCache[cacheKey] = cipher
			}
			ProxyServers.SrvCipher[i] = &ProxyServerCipher{server, cipher}
			i++
		}
	}
	ProxyServers.FailCnt = make([]int, len(ProxyServers.SrvCipher))
	for _, se := range ProxyServers.SrvCipher {
		log.Println("available remote server", se.Server)
	}
	return
}
