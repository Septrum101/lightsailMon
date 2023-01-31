package dns

import (
	"testing"
)

func TestLookupIP(t *testing.T) {
	c := New("tcp4", "https://dot.pub/dns-query")

	ips := c.LookupIP("qq.com")
	if len(ips) == 0 {
		t.Error("lookup DNS failure")
	}
	t.Log(ips)
}
