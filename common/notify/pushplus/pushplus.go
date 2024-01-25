package pushplus

import (
	"fmt"

	"github.com/go-resty/resty/v2"
)

func (p *PushPlus) Webhook(title string, content string) error {
	api := "https://www.pushplus.plus/send/"
	rtn := &pushPlusResp{}
	resp, err := resty.New().SetRetryCount(3).R().SetResult(rtn).SetBody(map[string]string{
		"token":   p.Token,
		"title":   title,
		"content": content,
	}).ForceContentType("application/json").Post(api)
	if err != nil {
		return err
	}

	switch rtn.Code {
	case 0:
		return fmt.Errorf("[PushPlus] %s", resp.String())
	case 200:
		return nil
	default:
		return fmt.Errorf("[PushPlus] %s", rtn.Msg)
	}
}
