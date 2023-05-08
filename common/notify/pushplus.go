package notify

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type PushPlus struct {
	Token string
}

func (p *PushPlus) Webhook(content string) error {
	api := "http://www.pushplus.plus/send/"
	u, _ := url.Parse(api)
	q := u.Query()
	q.Add("token", p.Token)
	q.Add("content", content)
	u.RawQuery = q.Encode()
	resp, err := http.Get(u.String())
	if err != nil {
		return err
	}

	b, _ := io.ReadAll(resp.Body)
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
