package flora

import (
	"github.com/oschwald/geoip2-golang"
	"log"
	"net"
	"strings"
)

var geoDB *geoip2.Reader

func loadGeoIP() {
	file := "./geoip.mmdb"
	db, err := geoip2.Open(file)
	// defer db.Close()
	if err != nil {
		log.Printf("Could not open GeoIP database\n")
	}
	// log.Println("GeoIP inited.")
	geoDB = db
}

func GeoIPString(ipaddr string) string {
	ip := net.ParseIP(ipaddr)
	return GeoIP(ip)
}

func GeoIPs(ips []net.IP) string {
	if len(ips) == 0 {
		return ""
	}

	return GeoIP(ips[0])
}

func GeoIP(ip net.IP) string {
	// log.Println("Lookup GEO IP", ip)
	country, err := geoDB.Country(ip)
	if err != nil {
		return ""
	}
	return strings.ToLower(country.Country.IsoCode)
}
