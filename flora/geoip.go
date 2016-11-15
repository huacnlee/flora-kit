package flora

import (
	"fmt"
	"github.com/oschwald/geoip2-golang"
	"net"
	"strings"
)

var geoDB *geoip2.Reader

func init() {
	file := "./geoip.mmdb"
	db, err := geoip2.Open(file)
	defer db.Close()
	if err != nil {
		fmt.Printf("Could not open GeoIP database\n")
	}
	geoDB = db
}

func GeoIP(ipaddr string) string {
	ip := net.ParseIP(ipaddr)
	country, err := geoDB.Country(ip)
	if err != nil {
		return ""
	}
	return strings.ToLower(country.Country.IsoCode)
}
