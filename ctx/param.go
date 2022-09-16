package ctx

type Values map[string][]string

func (c *Context) Param() Values {
	p := make(Values)
	for key, indexs := range c.ParamKeys {
		for _, i := range indexs {
			p[key] = append(p[key], c.ParamVals[i])
		}
	}

	return p
}

// func (p Values) Add(key string, value T) {
// 	p[key] = value
// }

// func (p Values) Del(key string) {
// 	delete(p, key)
// }

func (p Values) Get(key string) string {
	return ""
}

// func (p Values) Has(key string) bool {
// 	return reflect.ValueOf(p[key]).IsNil()
// }

// func (p Values) Set(key string, value T) {
// 	p[key] = value
// }
