package telegram

import (
	"fmt"
	"time"

	"github.com/go-resty/resty/v2"
	log "github.com/sirupsen/logrus"
)

func (t *Telegram) Webhook(title string, content string) error {
	api := fmt.Sprintf("https://%s/bot%s/sendMessage", t.ApiHost, t.Token)

	for i := 0; i < 3; i++ {
		_, err := resty.New().SetRetryCount(3).R().SetBody(map[string]any{
			"chat_id": t.ChatID,
			"text": fmt.Sprintf("#LightsailMon\nNode: %s\n%s",
				title,
				content,
			),
		}).Post(api)
		if err == nil {
			break
		}

		log.Debugf("[telegram] %v, attempt retry..(%d/3)", err, i+1)
		time.Sleep(time.Second * 5)
	}

	return nil
}
