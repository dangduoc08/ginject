package context

type WSPayload map[string]any

func (p WSPayload) Bind(s any) any {
	return BindStruct(p, s)
}
