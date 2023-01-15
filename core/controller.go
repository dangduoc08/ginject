package core

type Controller interface {
	Inject() Controller
}
