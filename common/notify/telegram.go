package notify

import (
	"fmt"
	"time"

	tg "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	log "github.com/sirupsen/logrus"
)

type Telegram struct {
	ChatID int64
	Token  string
}

func (t *Telegram) Webhook(title string, content string) error {
	log.SetLevel(log.DebugLevel)
	bot, err := tg.NewBotAPI(t.Token)
	if err != nil {
		return err
	}

	msg := tg.NewMessage(t.ChatID, fmt.Sprintf("LightsailMon\nNode: %s\n%s",
		title,
		content,
	))
	for i := 0; i < 3; i++ {
		if _, err = bot.Send(msg); err == nil {
			break
		}
		log.Debugf("[telegram] %v, attempt retry..(%d/3)", err, i+1)
		time.Sleep(time.Second * 5)
	}

	if err != nil {
		return err
	}

	return nil
}
