package ddns

import (
	"strings"
	"testing"

	"github.com/Septrum101/lightsailMon/common/ddns/google"
)

func TestGoogle(t *testing.T) {
	g, err := google.New(map[string]string{
		strings.ToLower("GOOGLEDOMAIN_USERNAME"): "username",
		strings.ToLower("GOOGLEDOMAIN_PASSWORD"): "password",
	}, "subdomain.yourdomain.com")
	if err != nil {
		t.Error(err)
	}

	err = g.AddUpdateDomainRecords("tcp4", "1.2.3.4")
	if err != nil {
		t.Error(err)
	}
}
