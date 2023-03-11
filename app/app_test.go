package app

import (
	"testing"
)

func TestCheckConnection(t *testing.T) {
	_, err := CheckConnection("1.1.1.1", 80, 15, "tcp4")
	if err != nil {
		t.Log(err)
	}
}
