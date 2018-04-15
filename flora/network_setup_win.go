package flora

import (
	"log"
	"os/exec"
)

type windows struct {
	address string
}

const (
	cmdRegistry         = `reg`
	cmdRegistryAdd      = `add`
	internetSettingsKey = `HKEY_CURRENT_USER\Software\Microsoft\Windows\CurrentVersion\Internet Settings`
	keyProxyEnable      = `ProxyEnable`
	keyProxyServer      = `ProxyServer`
	dataTypeDWord       = `REG_DWORD`
	dataTypeRegSZ       = `REG_SZ`
)

func (w *windows) TurnOnGlobProxy() {
	c := exec.Command(cmdRegistry, cmdRegistryAdd, internetSettingsKey, `/v`, keyProxyEnable, `/t`, dataTypeDWord, `/d`, `1`, `/f`)
	var err error
	if _, err = c.CombinedOutput(); err != nil {
		log.Printf("enable windows proxy has error %s", err)
	}
	c = exec.Command(cmdRegistry, cmdRegistryAdd, internetSettingsKey, `/v`, keyProxyServer, `/t`, dataTypeRegSZ, `/d`, w.address, `/f`)
	if _, err = c.CombinedOutput(); err != nil {
		log.Printf("Windows global proxy settings has error %s , Try to set it manually ", err)
	}
	if nil == err {
		log.Print("Windows global proxy settings are successful ，Please use after 2 minutes ...")
	}
}

// TurnOffSystemProxy
func (w *windows) TurnOffGlobProxy() {
	var err error
	c := exec.Command(cmdRegistry, cmdRegistryAdd, internetSettingsKey, `/v`, keyProxyEnable, `/t`, dataTypeDWord, `/d`, `0`, `/f`)
	if _, err = c.CombinedOutput(); err != nil {
		log.Printf("disable windows proxy has error %s", err)
	}
	if nil == err {
		log.Print("disable windows proxy settings  are successful ...")
	}
}
