package flora

import (
	"os/signal"
	"syscall"
	"os"
	"log"
	"runtime"
	"time"
)

type SystemProxySettings interface {
	TurnOnGlobProxy()
	TurnOffGlobProxy()
}

var sigs = make(chan os.Signal, 1)

func resetProxySettings(proxySettings SystemProxySettings) {
	for {
		select {
		case <-sigs:
			log.Print("Flora-kit is shutdown now ...")
			proxySettings.TurnOffGlobProxy()
			time.Sleep(time.Duration(2000))
			os.Exit(0)
		}
	}
}

func initProxySettings(bypass []string, addr string)  {
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	var proxySettings SystemProxySettings
	if runtime.GOOS == "windows" {
		w := &windows{addr}
		proxySettings = w
	} else if runtime.GOOS == "darwin" {
		d := &darwin{bypass,addr}
		proxySettings = d
	}
	proxySettings.TurnOnGlobProxy()
	go resetProxySettings(proxySettings)
}
