package notify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type PushPlus struct {
	Token string
}

func (p *PushPlus) Webhook(title string, content string) error {
	api := "https://www.pushplus.plus/send/"
	pl := bytes.Buffer{}
	b, _ := json.Marshal(map[string]string{
		"token":   p.Token,
		"title":   title,
		"content": content,
	})
	pl.Write(b)
	resp, err := http.Post(api, "application/json", &pl)
	if err != nil {
		return err
	}

	b, _ = io.ReadAll(resp.Body)
	defer resp.Body.Close()

	var buf map[string]any
	if err := json.Unmarshal(b, &buf); err != nil {
		return err
	}

	if v, ok := buf["code"]; ok {
		if v.(float64) != 200 {
			return fmt.Errorf("[PushPlus] %s", buf["msg"])
		}
	} else {
		return fmt.Errorf("[PushPlus] %v", buf)
	}

	return nil
}
