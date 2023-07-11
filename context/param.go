package context

type Param map[string][]string

type Parameter interface {
	Get(string) string
	Set(string, string)
	Add(string, string)
	Del(string)
	Has(string) bool
}

func (c *Context) Param() Param {
	if c.param != nil {
		return c.param
	}

	c.param = make(Param)
	for key, indexs := range c.ParamKeys {
		for _, i := range indexs {
			c.param[key] = append(c.param[key], c.ParamValues[i])
		}
	}

	return c.param
}

func (p Param) Get(key string) string {
	if p == nil {
		return ""
	}
	ps := p[key]
	if len(ps) == 0 {
		return ""
	}
	return ps[0]
}

func (p Param) Set(key, value string) {
	p[key] = []string{value}
}

func (p Param) Add(key, value string) {
	p[key] = append(p[key], value)
}

func (p Param) Del(key string) {
	delete(p, key)
}

func (p Param) Has(key string) bool {
	_, ok := p[key]
	return ok
}
