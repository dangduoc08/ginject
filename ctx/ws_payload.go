package ctx

type WSPayload map[string]any

func (p WSPayload) Bind(s any) (any, []FieldLevel) {
	return BindStruct(p, &[]FieldLevel{}, s, "", "")
}
