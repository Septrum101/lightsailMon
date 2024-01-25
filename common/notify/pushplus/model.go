package pushplus

type PushPlus struct {
	Token string
}

type pushPlusResp struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data string `json:"data"`
}
