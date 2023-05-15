package notify

type Notify interface {
	Webhook(title string, content string) error
}
