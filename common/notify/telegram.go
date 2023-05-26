package notify

import (
	"fmt"

	tg "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Telegram struct {
	ChatID int64
	Token  string
}

func (t *Telegram) Webhook(title string, content string) error {
	bot, err := tg.NewBotAPI(t.Token)
	if err != nil {
		return err
	}

	msg := tg.NewMessage(t.ChatID, fmt.Sprintf("LightsailMon\nNode: %s\n%s",
		title,
		content,
	))
	if _, err = bot.Send(msg); err != nil {
		return err
	}
	return nil
}
