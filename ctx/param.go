package ctx

type Param map[string][]string

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

func (p Param) Get(k string) string {
	if p == nil {
		return ""
	}
	ps := p[k]
	if len(ps) == 0 {
		return ""
	}
	return ps[0]
}

func (p Param) Set(k, v string) {
	p[k] = []string{v}
}

func (p Param) Add(k, v string) {
	p[k] = append(p[k], v)
}

func (p Param) Del(k string) {
	delete(p, k)
}

func (p Param) Has(k string) bool {
	_, ok := p[k]
	return ok
}

func (p Param) Bind(s any) any {
	return BindStrArr(p, s)
}
