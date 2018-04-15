package main_test

import (
	"flora-kit/flora"
	"os"
	"testing"
)

func TestGeoIP(t *testing.T) {
	p, _ := os.Getwd()
	flora.LoadConfig("", p+"/geoip.mmdb")

	if flora.GeoIPString("121.0.29.91") != "cn" {
		t.Errorf("121.0.29.91 should be cn")
	}

	if flora.GeoIPString("218.253.0.89") != "hk" {
		t.Errorf("218.253.0.89 should be hk")
	}

	if flora.GeoIPString("218.176.242.11") != "jp" {
		t.Errorf("218.176.242.11 should be jp")
	}

	if flora.GeoIPString("8.8.8.8") != "us" {
		t.Errorf("218.176.242.11 should be jp")
	}
}
