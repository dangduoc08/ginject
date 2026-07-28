package common

type Publisher interface {
	Publish(topic string, payload ...any) error
}
