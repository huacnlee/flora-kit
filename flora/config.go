package flora

import (
	"fmt"
	"github.com/go-ini/ini"
	ss "github.com/shadowsocks/shadowsocks-go/shadowsocks"
	"log"
	"os"
	"strings"
)

const (
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

var Config ss.Config
var iniConfig *ini.File

func init() {
	Config.LocalPort = 7657

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
	for _, key := range iniConfig.Section("Proxy").Keys() {
		var proxys = readArrayLine(key.String())
		// ShadowSocks Proxys
		if proxys[0] == "custom" || proxys[0] == "shadowsocks" {
			var server = strings.Join(proxys[1:3], ":")
			var serverInfo = []string{server, proxys[4], proxys[3]}
			Config.ServerPassword = append(Config.ServerPassword, serverInfo)
		}
	}

	if Config.Method == "" {
		Config.Method = "aes-256-cfb"
	}
	if len(Config.ServerPassword) == 0 {
		if !enoughSSOptions(&Config) {
			fmt.Fprintln(os.Stderr, "must specify server address, password and both server/local port")
			os.Exit(1)
		}
	} else {
		if Config.LocalPort == 0 {
			fmt.Fprintln(os.Stderr, "must specify local port")
			os.Exit(1)
		}
	}
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
		out[i] = strings.Trim(str, " ")
		out[i] = strings.Trim(str, "\t")
		out[i] = strings.Trim(str, "\n")
	}
	return out
}

func RuleOfHost(host string) *DomainRule {
	for _, rule := range ruleSuffixDomains {
		if strings.HasSuffix(host, rule.S) {
			return rule
		}
	}

	for _, rule := range rulePrefixDomains {
		if strings.HasPrefix(host, rule.S) {
			return rule
		}
	}

	for _, rule := range ruleKeywordDomains {
		if strings.Contains(host, rule.S) {
			return rule
		}
	}
	return &DomainRule{S: "", T: RULE_DIRECT}
}
