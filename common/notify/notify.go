package notify

type Notify interface {
	Webhook(content string) error
}
