package app

import (
	"testing"
)

func TestCheckConnection(t *testing.T) {
	err := CheckConnection("1.1.1.1", 80, 15, "tcp4")
	if err != nil {
		t.Log(err)
	}
}
