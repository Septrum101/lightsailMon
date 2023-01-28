package app

import (
	"testing"
)

func TestLookupIP(t *testing.T) {
	ips := LookupIP("qq.com")
	if len(ips) == 0 {
		t.Error("lookup DNS failure")
	}
	t.Log(ips)
}
