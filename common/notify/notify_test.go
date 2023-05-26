package notify

import (
	"testing"
)

func TestTelegram_Webhook(t *testing.T) {
	tg := Telegram{
		ChatID: 123,
		Token:  "YOUR_TOKEN",
	}
	tg.Webhook("node1.test.com", "This is test message")
}
