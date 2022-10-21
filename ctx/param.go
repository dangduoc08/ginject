package ctx

type Values map[string][]string

type Parameter interface {
	Get(string) string
	Set(string, string)
	Add(string, string)
	Del(string)
	Has(string) bool
}

func (c *Context) Param() Values {
	if c.param != nil {
		return c.param
	}

	p := make(Values)
	for key, indexs := range c.paramKeys {
		for _, i := range indexs {
			p[key] = append(p[key], c.paramVals[i])
		}
	}

	c.param = p
	return p
}

func (v Values) Get(key string) string {
	if v == nil {
		return ""
	}
	vs := v[key]
	if len(vs) == 0 {
		return ""
	}
	return vs[0]
}

func (v Values) Set(key, value string) {
	v[key] = []string{value}
}

func (v Values) Add(key, value string) {
	v[key] = append(v[key], value)
}

func (v Values) Del(key string) {
	delete(v, key)
}

func (v Values) Has(key string) bool {
	_, ok := v[key]
	return ok
}
