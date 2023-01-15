package core

type Provider interface {
	Inject() Provider
}
