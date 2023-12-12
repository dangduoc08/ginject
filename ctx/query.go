package ctx

type Query map[string][]string

func (c *Context) Query() Query {
	if c.query != nil {
		return c.query
	}
	c.query = Query(c.URL.Query())

	return c.query
}

func (q Query) Get(k string) string {
	qs := q[k]
	if len(qs) == 0 {
		return ""
	}
	return qs[0]
}

func (q Query) Set(k, v string) {
	q[k] = []string{v}
}

func (q Query) Add(k, v string) {
	q[k] = append(q[k], v)
}

func (q Query) Del(k string) {
	delete(q, k)
}

func (q Query) Has(k string) bool {
	_, ok := q[k]
	return ok
}

func (q Query) Bind(s any) any {
	return BindStrArr(q, s)
}
