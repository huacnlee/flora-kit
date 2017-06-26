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
	"math/rand"
	"sync"
)

const (
	VERSION     = "0.1.1"
	SOCKS_PORT  = 1080
	RULE_REJECT = "REJECT"
	RULE_DIRECT = "DIRECT"
)

type HostRule struct {
	Match  string
	Action string
}

var ruleSuffixDomains = []*HostRule{}
var rulePrefixDomains = []*HostRule{}
var ruleKeywordDomains = []*HostRule{}
var ruleGeoIP = []*HostRule{}
var ruleFinal *HostRule
var failMapLock *sync.RWMutex
var iniConfig *ini.File

type ProxyServerCipher struct {
	ProxyType         string
	Server            string
	Effective         bool
	ShadowSocksCipher *ss.Cipher
}

type ProxyConfig struct {
	Name              string
	Type              string
	ShadowSocksConfig []string
}

var ProxyServers struct {
	ListenAddress  string
	LocalSocksPort int
	SrvCipher      map[string]*ProxyServerCipher
	SrvCipherGroup map[string][]*ProxyServerCipher
	GetCipher      func(name string) *ProxyServerCipher
	FailCipher     func(name string) int
	failCipher     *SyncMap // failed connection count
}

func LoadConfig(cfgFile string, geoFile string) {
	sep := string(os.PathSeparator)
	pwd, _ := os.Getwd()

	var defaultCfgName = strings.Join([]string{pwd, "flora.default.conf"}, sep)
	var userConfigFilename = strings.Join([]string{pwd, "flora.user.conf"}, sep)
	var geoFilename = geoFile
	var err error
	if _, err := os.Stat(geoFilename); nil != err && os.IsNotExist(err) {
		geoFilename = strings.Join([]string{pwd, "geoip.mmdb"}, sep)
	}
	var iniOpts = ini.LoadOptions{
		AllowBooleanKeys: true,
		Loose:            true,
		Insensitive:      true,
	}
	iniConfig, err = ini.LoadSources(iniOpts, cfgFile, defaultCfgName, userConfigFilename)

	if err != nil {
		panic(fmt.Sprintf("Config file %v not found, or have error: \n\t%v", geoFile, err))
	}
	loadProxyGroup()
	loadGeoIP(geoFilename)
	loadGeneral()
	loadRules()
	SetSocksFirewallProxy()
}

// [General] section
func loadGeneral() {
	section := iniConfig.Section("General")

	bypassDomains := []string{}
	if section.HasKey("skip-proxy") {
		bypassDomains = append(bypassDomains, readArrayLine(section.Key("skip-proxy").String())...)
	}
	if section.HasKey("bypass-tun") {
		bypassDomains = append(bypassDomains, readArrayLine(section.Key("bypass-tun").String())...)
	}
	if section.HasKey("socks-port") {
		port, err := strconv.Atoi(section.Key("socks-port").String())
		if nil != err {
			port = SOCKS_PORT
		} else {
			ProxyServers.LocalSocksPort = port
		}
	}
	if section.Haskey("interface") {
		ipStr := section.Key("interface").String()
		addr := net.ParseIP(ipStr)
		if nil == addr {
			ProxyServers.ListenAddress = "127.0.0.1"
		} else {
			ProxyServers.ListenAddress = ipStr
		}
	}
	SetProxyBypassDomains(bypassDomains)
}

// [Proxy] Section
func loadProxy() map[string]ProxyConfig {
	serverMapping := make(map[string]ProxyConfig)
	section := iniConfig.Section("Proxy")
	for _, name := range section.KeyStrings() {
		v, _ := section.GetKey(name)
		var proxyStrCfg = readArrayLine(v.String())
		var proxy = ProxyConfig{Type: proxyStrCfg[0], Name: name}
		// ShadowSocks Proxy
		if len(proxyStrCfg) > 0 && (proxyStrCfg[0] == "custom" || proxyStrCfg[0] == "shadowsocks") {
			//[ip:port,password,method]
			var server = strings.Join(proxyStrCfg[1:3], ":")
			var serverInfo = []string{server, proxyStrCfg[4], proxyStrCfg[3]}
			proxy.ShadowSocksConfig = serverInfo
		}
		serverMapping[name] = proxy
	}
	return serverMapping
}

//[Proxy Group] Section
func loadProxyGroup() {
	const maxFailCnt = 30
	srvCipherMap := initProxyServerConfig()
	section := iniConfig.Section("Proxy Group")
	ProxyServers.SrvCipherGroup = make(map[string][]*ProxyServerCipher)
	for _, key := range section.KeyStrings() {
		v, _ := section.GetKey(key)
		groupName := strings.ToUpper(key)
		proxyArr := readArrayLine(v.String())
		proxyItems := make([]*ProxyServerCipher, len(proxyArr)-1)
		//🚀 Proxy = select, 🌞 Line
		if len(proxyArr) > 1 {
			for i, p := range proxyArr[1:] {
				proxyName := strings.ToUpper(p)
				proxyItems[i] = srvCipherMap[proxyName]
			}
		}
		ProxyServers.SrvCipherGroup[groupName] = proxyItems
	}

	ProxyServers.SrvCipher = srvCipherMap
	ProxyServers.failCipher = NewSyncMap()
	ProxyServers.FailCipher = func(name string) int {
		var cnt int
		failMapLock.Lock()
		itf := ProxyServers.failCipher.Get(name)
		defer failMapLock.Unlock()
		if nil != itf {
			cnt = itf.(int)
		}
		cnt ++
		ProxyServers.failCipher.Set(name, cnt)
		return cnt
	}

	ProxyServers.GetCipher = func(name string) *ProxyServerCipher {
		var cnt int
		itf := ProxyServers.failCipher.Get(name)
		if nil != itf {
			cnt = itf.(int)
			if cnt >= maxFailCnt {
				log.Printf("Proxy Server [%s] connect exceeds the maximum number of failures ", name)
				os.Exit(1)
			}
		}

		svrCipher, ok := ProxyServers.SrvCipher[name]
		if !ok {
			group := ProxyServers.SrvCipherGroup[name]
			eff := []*ProxyServerCipher{}
			for _, s := range group {
				e := s.Effective
				if e {
					eff = append(eff, s)
				}
			}
			return eff[rand.Intn(len(eff))]
		}
		return svrCipher
	}
}

func initProxyServerConfig() map[string]*ProxyServerCipher {
	hasPort := func(s string) bool {
		_, port, err := net.SplitHostPort(s)
		if err != nil {
			return false
		}
		return port != ""
	}
	proxySvrs := loadProxy()
	cipherCache := make(map[string]*ProxyServerCipher)
	for key, val := range proxySvrs {
		svrName := strings.ToUpper(strings.TrimSpace(key))
		serverCipher := ProxyServerCipher{ProxyType: val.Type, Effective: true}
		if val.Type == "custom" || val.Type == "shadowsocks" {
			serverInfo := val.ShadowSocksConfig
			server := serverInfo[0]
			passwd := serverInfo[1]
			encmethod := ""
			if len(serverInfo) == 3 {
				encmethod = serverInfo[2]
			}
			if !hasPort(server) {
				log.Printf("no port for server %s\n", server)
				ProxyServers.failCipher.Set(svrName, 0)
				continue
			}
			cipher, err := ss.NewCipher(encmethod, passwd)
			if err != nil {
				log.Printf("Failed generating ciphers %s\n", err)
				ProxyServers.failCipher.Set(svrName, 0)
				continue
			}
			serverCipher.Server = server
			serverCipher.ShadowSocksCipher = cipher
		}
		cipherCache[svrName] = &serverCipher
	}
	return cipherCache
}

// 载入 [Rule]
func loadRules() {
	for _, key := range iniConfig.Section("Rule").KeyStrings() {
		if strings.HasPrefix(key, "//") {
			continue
		}
		var (
			items    = readArrayLine(key)
			ruleType string
		)
		if len(items) >= 3 {
			switch items[2] {
			case "direct":
				ruleType = RULE_DIRECT
			case "reject":
				ruleType = RULE_REJECT
			default:
				ruleType = strings.ToUpper(items[2])
			}
		}
		ruleName := strings.ToLower(items[0])
		switch ruleName {
		case "domain-suffix":
			ruleSuffixDomains = append(ruleSuffixDomains, &HostRule{Match: items[1], Action: ruleType})
		case "domain-prefix":
			rulePrefixDomains = append(rulePrefixDomains, &HostRule{Match: items[1], Action: ruleType})
		case "domain-keyword":
			ruleKeywordDomains = append(ruleKeywordDomains, &HostRule{Match: items[1], Action: ruleType})
		case "geoip":
			ruleGeoIP = append(ruleGeoIP, &HostRule{Match: items[1], Action: ruleType})
		case "final":
			ruleFinal = &HostRule{Match: "final", Action: strings.ToUpper(items[1])}
		}
	}
}

func readArrayLine(source string) []string {
	out := strings.Split(source, ",")
	for i, str := range out {
		out[i] = strings.TrimSpace(str)
	}
	return out
}

func RuleOfHost(host string) (*HostRule) {
	hostParts := strings.Split(host, ":")
	domain := strings.ToLower(hostParts[0])
	for _, rule := range ruleSuffixDomains {
		if strings.HasSuffix(domain, rule.Match) {
			return rule
		}
	}
	for _, rule := range rulePrefixDomains {
		if strings.HasPrefix(domain, rule.Match) {
			return rule
		}
	}

	for _, rule := range ruleKeywordDomains {
		if strings.Contains(domain, rule.Match) {
			return rule
		}
	}
	ips := resolveRequestIPAddr(host)
	if nil != ips {
		country := strings.ToLower(GeoIPs(ips))
		log.Println("Found ip geo", country)
		for _, rule := range ruleGeoIP {
			if len(country) != 0 && strings.ToLower(rule.Match) == country {
				return rule
			}
		}
	}
	if nil != ruleFinal {
		return ruleFinal
	} else {
		return &HostRule{Match: "", Action: RULE_DIRECT}
	}
}

func resolveRequestIPAddr(host string) []net.IP {
	var (
		ips []net.IP
		err error
	)
	ip := net.ParseIP(host)
	if nil == ip {
		ips, err = net.LookupIP(host)
		if err != nil || len(ips) == 0 {
			return nil
		}
	} else {
		ips = []net.IP{ip}
	}
	return ips
}
