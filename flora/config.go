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
	VERSION     = "0.1.1"
	SOCKS_PORT  = 7657
	RULE_REJECT = "REJECT"
	RULE_DIRECT = "DIRECT"
	RULE_PROXY  = "PROXY"
)

type DomainRule struct {
	S string
	T string
}

var ruleSuffixDomains = []*DomainRule{}
var rulePrefixDomains = []*DomainRule{}
var ruleKeywordDomains = []*DomainRule{}
var ruleGeoIP = &DomainRule{}

var iniConfig *ini.File
var debug ss.DebugLog

type ProxyServerCipher struct {
	Server string
	Cipher *ss.Cipher
}

type ProxyConfig struct {
	Name   string
	Type   string
	Config []string
}

var ProxyServers struct {
	SrvCipher map[string][]*ProxyServerCipher
	FailCnt   map[string]int // failed connection count
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
	var ssConfig = ss.Config{}

	loadGeoIP(geoFilename)
	loadGeneral(&ssConfig)
	loadRules()

	SetSocksFirewallProxy()

	debug.Println("104.244.42.129", GeoIPString("104.244.42.129"))
	debug.Println(RuleOfHost("www.twitter.com"))
}

// [General] section
func loadGeneral(ssCfg *ss.Config) {
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
			ssCfg.LocalPort = SOCKS_PORT
		}
		ssCfg.LocalPort = port
	} else {
		ssCfg.LocalPort = SOCKS_PORT
	}

	SetProxyBypassDomains(bypassDomains)
}

// [Proxy] Section
func loadProxy() map[string]ProxyConfig {
	serverMapping := make(map[string]ProxyConfig)
	section := iniConfig.Section("Proxy")
	for _, key := range section.KeyStrings() {
		v, _ := section.GetKey(key)
		var proxyStrCfg = readArrayLine(v.String())
		var proxy = ProxyConfig{Type: proxyStrCfg[0], Name: key}
		// ShadowSocks Proxy
		if len(proxyStrCfg) > 0 && (proxyStrCfg[0] == "custom" || proxyStrCfg[0] == "shadowsocks") {
			//[ip:port,password,method]
			var server = strings.Join(proxyStrCfg[1:3], ":")
			var serverInfo = []string{server, proxyStrCfg[4], proxyStrCfg[3]}
			proxy.Config = serverInfo
		}
		serverMapping[key] = proxy
	}
	return serverMapping
}

// Proxy -> name -> proxyServerList index
//
//
func loadProxyGroup() {
	proxyList := initProxyServerConfig()
	section := iniConfig.Section("Proxy Group")
	ProxyServers.SrvCipher = make(map[string][]*ProxyServerCipher)
	for _, key := range section.KeyStrings() {
		v, _ := section.GetKey(key)
		proxyArr := readArrayLine(v.String())
		//t := proxyArr[0]
		proxyItems := make([]*ProxyServerCipher, len(proxyArr)-1)
		for _, p := range proxyArr[1:] {
			proxyItems = append(proxyItems, proxyList[p])
		}
		ProxyServers.SrvCipher[key] = proxyItems
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
	for k, v := range proxySvrs {
		serverInfo := v.Config
		server := serverInfo[0]
		passwd := serverInfo[1]
		encmethod := ""
		if len(serverInfo) == 3 {
			encmethod = serverInfo[2]
		}
		if !hasPort(server) {
			log.Printf("no port for server %s\n", server)
			continue
		}
		cipher, err := ss.NewCipher(encmethod, passwd)
		if err != nil {
			log.Printf("Failed generating ciphers %s\n", err)
			continue
		}
		cipherCache[k] = &ProxyServerCipher{server, cipher}
	}
	return cipherCache
}

// 载入 [Rule]
func loadRules() {
	for _, key := range iniConfig.Section("Rule").KeyStrings() {
		var items = readArrayLine(key)
		var ruleType = RULE_DIRECT
		if len(items) >= 3 {
			switch items[2] {
			case "direct":
				ruleType = RULE_DIRECT
			case "reject":
				ruleType = RULE_REJECT
			default:
				ruleType = items[2]
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

func RuleOfHost(host string) (result *DomainRule) {
	result = &DomainRule{S: "", T: RULE_DIRECT}
	hostParts := strings.Split(host, ":")
	domain := hostParts[0]

	for _, rule := range ruleSuffixDomains {
		if strings.HasSuffix(domain, rule.S) {
			result = rule
			return
		}
	}

	for _, rule := range rulePrefixDomains {
		if strings.HasPrefix(domain, rule.S) {
			result = rule
			return
		}
	}

	for _, rule := range ruleKeywordDomains {
		if strings.Contains(domain, rule.S) {
			result = rule
			return
		}
	}

	ips, err := net.LookupIP(host)
	if err != nil || len(ips) == 0 {
		return
	}

	country := GeoIPs(ips)
	log.Println("Found ip geo", country)
	if len(country) != 0 && ruleGeoIP.S == country {
		result = ruleGeoIP
		return
	}

	return
}
