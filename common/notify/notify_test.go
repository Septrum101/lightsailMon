package notify

import (
	"testing"

	"github.com/thank243/lightsailMon/common/notify/pushplus"
	"github.com/thank243/lightsailMon/common/notify/telegram"
)

func TestTelegram_Webhook(t *testing.T) {
	tg := telegram.Telegram{
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
	pp := pushplus.PushPlus{Token: "YOUR_TOKEN"}
	err := pp.Webhook("node1.test.com", "This is test message")
	if err != nil {
		t.Error(err)
		return
	}
}
