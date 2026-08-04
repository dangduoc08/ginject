package memorybroker

func (b *MemoryBroker) callHandler(h MessageHandler, msg *Message) {
	if !b.cfg.RecoverPanics {
		h(msg)
		return
	}
	defer func() {
		if r := recover(); r != nil {
			b.runOnPanic(msg, r)
		}
	}()
	h(msg)
}

func (b *MemoryBroker) runOnPanic(msg *Message, r any) {
	if b.cfg.OnPanic == nil {
		return
	}
	defer func() { _ = recover() }()
	b.cfg.OnPanic(msg, r)
}

func (b *MemoryBroker) runBeforePublish(topic string, payload any) {
	defer func() { _ = recover() }()
	b.cfg.BeforePublish(topic, payload)
}

func (b *MemoryBroker) runAfterPublish(topic string, payload any, err error) {
	defer func() { _ = recover() }()
	b.cfg.AfterPublish(topic, payload, err)
}

func (b *MemoryBroker) runBeforeDispatch(msg *Message, i int) {
	defer func() { _ = recover() }()
	b.cfg.BeforeDispatch(msg, i)
}

func (b *MemoryBroker) runAfterDispatch(msg *Message, i int) {
	defer func() { _ = recover() }()
	b.cfg.AfterDispatch(msg, i)
}
