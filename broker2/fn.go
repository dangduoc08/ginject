package broker2

func callHandler(h MessageHandler, msg *Message) {
	defer func() { _ = recover() }()
	h(msg)
}
