package main_test

import (
	"github.com/huacnlee/flora-kit/flora"
	"testing"
)

func TestGeoIP(t *testing.T) {
	if flora.GeoIP("121.0.29.91") != "cn" {
		t.Errorf("121.0.29.91 should be cn")
	}

	if flora.GeoIP("218.253.0.89") != "hk" {
		t.Errorf("218.253.0.89 should be hk")
	}

	if flora.GeoIP("218.176.242.11") != "jp" {
		t.Errorf("218.176.242.11 should be jp")
	}
}
