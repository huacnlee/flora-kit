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
)

const (
	DEFAULT_SOCKS_PORT = 1080
)

type ProxyConfig struct {
	SurgeConfig    *ini.File
	GeoDbPath      string
	LocalSocksPort int
	LocalHost      string
	proxyServer    map[string]ProxyServer
	proxyGroup     map[string]*proxyGroup

	bypassDomains      []interface{}
	ruleSuffixDomains  []*Rule
	rulePrefixDomains  []*Rule
	ruleKeywordDomains []*Rule
	ruleUserAgent      []*Rule
	ruleGeoIP          []*Rule
	ruleFinal          *Rule
}
type proxyGroup struct {
	mode         string
	proxyServers []ProxyServer
}

func LoadConfig(cfgFile string, geoFile string) (*ProxyConfig) {
	proxyConfig := ProxyConfig{}
	var iniOpts = ini.LoadOptions{
		AllowBooleanKeys: true,
		Loose:            true,
		Insensitive:      true,
	}
	sep := string(os.PathSeparator)
	pwd, _ := os.Getwd()
	var geoFilename = geoFile
	var err error
	var defaultCfgName = strings.Join([]string{pwd, "flora.default.conf"}, sep)
	var userConfigFilename = strings.Join([]string{pwd, "flora.user.conf"}, sep)
	var speCfgName string
	if _, err := os.Stat(cfgFile); nil != err && os.IsNotExist(err) {
		speCfgName = strings.Join([]string{pwd, cfgFile}, sep)
	}
	if _, err := os.Stat(geoFilename); nil != err && os.IsNotExist(err) {
		geoFilename = strings.Join([]string{pwd, "geoip.mmdb"}, sep)
	}

	proxyConfig.SurgeConfig, err = ini.LoadSources(iniOpts, speCfgName, defaultCfgName, userConfigFilename)

	if err != nil {
		panic(fmt.Sprintf("Config file %v not found, or have error: \n\t%v", cfgFile, err))
	}
	loadGeneral(&proxyConfig)
	loadProxy(&proxyConfig)
	loadProxyGroup(&proxyConfig)
	loadGeoIP(geoFilename)
	loadRules(&proxyConfig)
	SetSocksFirewallProxy()

	return &proxyConfig
}

// [General] section
func loadGeneral(config *ProxyConfig) {
	section := config.SurgeConfig.Section("General")
	bypassDomains := []string{}
	if section.HasKey("skip-proxy") {
		bypassDomains = append(bypassDomains, readArrayLine(section.Key("skip-proxy").String())...)
	}
	if section.HasKey("bypass-tun") {
		bypassDomains = append(bypassDomains, readArrayLine(section.Key("bypass-tun").String())...)
	}
	config.LocalSocksPort = DEFAULT_SOCKS_PORT
	if section.HasKey("socks-port") {
		port, err := strconv.Atoi(section.Key("socks-port").String())
		if nil == err {
			config.LocalSocksPort = port
		}
	}
	config.LocalHost = "127.0.0.1"
	if section.Haskey("interface") {
		ipStr := section.Key("interface").String()
		addr := net.ParseIP(ipStr)
		if nil != addr {
			config.LocalHost = ipStr
		}
	}

	//load bypass
	config.bypassDomains = make([]interface{}, len(bypassDomains))
	for i, v := range bypassDomains {
		ip := net.ParseIP(v)
		if nil != ip {
			config.bypassDomains[i] = ip
		}else if _,n,err := net.ParseCIDR(v) ; err == nil {
			config.bypassDomains[i] = n
		}else{
			config.bypassDomains[i] = v
		}

	}

	SetProxyBypassDomains(bypassDomains)
}

// [Proxy] Section
func loadProxy(config *ProxyConfig) {
	config.proxyServer = make(map[string]ProxyServer)
	section := config.SurgeConfig.Section("Proxy")
	for _, name := range section.KeyStrings() {
		v, _ := section.GetKey(name)
		var proxyStrCfg = readArrayLine(v.String())
		serverType := strings.ToLower(proxyStrCfg[0])
		var proxy ProxyServer
		if serverType == ServerTypeShadowSocks || serverType == ServerTypeCustom {
			//[ip:port,password,method]
			if len(proxyStrCfg) > 1 {
				c, err := ss.NewCipher(proxyStrCfg[3], proxyStrCfg[4])
				if nil != err {
					log.Printf("Loading shadowsocks proxy server %s has error ", name)
					continue
				}
				proxy = NewShadowSocks(strings.Join(proxyStrCfg[1:3], ":"), c)
			}

		} else if serverType == ServerTypeDirect {
			proxy = NewDirect()
		} else if serverType == ServerTypeReject {
			proxy = NewReject()
		}
		if nil != proxy {
			log.Printf("Loading proxy server %s done. ", name)
			config.proxyServer[name] = proxy
		}
	}

}

func (c *ProxyConfig) GetProxyServer(action string) (ProxyServer) {
	const maxFailCnt = 30
	var server ProxyServer
	var ok bool
	server, ok = c.proxyServer[action]
	if !ok {
		group, ok := c.proxyGroup[action]
		if ok {
			for _, s := range group.proxyServers {
				eff := []ProxyServer{}
				if s.FailCount() < maxFailCnt {
					eff = append(eff, s)
				}
				l := len(eff)
				if l > 0 {
					return eff[rand.Intn(l)]
				}
			}
		} else {
			server = NewDirect()
		}
	}
	return server
}

//[Proxy Group] Section
func loadProxyGroup(config *ProxyConfig) {
	section := config.SurgeConfig.Section("Proxy Group")
	config.proxyGroup = make(map[string]*proxyGroup)
	for _, groupName := range section.KeyStrings() {
		v, _ := section.GetKey(groupName)
		proxyArr := readArrayLine(v.String())
		//🚀 Proxy = select, 🌞 Line
		if len(proxyArr) > 1 {
			groupItems := proxyGroup{mode: proxyArr[0]}
			servers := make([]ProxyServer, len(proxyArr)-1)
			for i, p := range proxyArr[1:] {
				proxyName := strings.ToLower(p)
				servers[i] = config.proxyServer[proxyName]
			}
			groupItems.proxyServers = servers
			config.proxyGroup[groupName] = &groupItems
		}
	}

}

//[Rule] Section
func loadRules(config *ProxyConfig) {
	for _, key := range config.SurgeConfig.Section("Rule").KeyStrings() {
		if strings.HasPrefix(key, "//") {
			continue
		}
		items := readArrayLine(key)
		ruleName := strings.ToLower(items[0])
		switch ruleName {
		case "user-agent":
			config.ruleUserAgent = append(config.ruleUserAgent, &Rule{Match: items[1], Action: strings.ToLower(items[2])})
		case "domain-suffix":
			config.ruleSuffixDomains = append(config.ruleSuffixDomains, &Rule{Match: items[1], Action: strings.ToLower(items[2])})
		case "domain-prefix":
			config.rulePrefixDomains = append(config.rulePrefixDomains, &Rule{Match: items[1], Action: strings.ToLower(items[2])})
		case "domain-keyword":
			config.ruleKeywordDomains = append(config.ruleKeywordDomains, &Rule{Match: items[1], Action: strings.ToLower(items[2])})
		case "geoip":
			config.ruleGeoIP = append(config.ruleGeoIP, &Rule{Match: items[1], Action: strings.ToLower(items[2])})
		case "final":
			config.ruleFinal = &Rule{Match: "final", Action: strings.ToUpper(items[1])}
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
