package notify

import (
	"testing"
)

func TestTelegram_Webhook(t *testing.T) {
	tg := Telegram{
		ChatID: 123,
		Token:  "YOUR_TOKEN",
	}
	err := tg.Webhook("node1.test.com", "This is test message")
	if err != nil {
		t.Error(err)
		return
	}
}

func TestPushPlus_Webhook(t *testing.T) {
	pp := PushPlus{Token: "YOUR_TOKEN"}
	err := pp.Webhook("node1.test.com", "This is test message")
	if err != nil {
		t.Error(err)
		return
	}
}
