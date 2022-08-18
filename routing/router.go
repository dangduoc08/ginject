package routing

type routerData struct {
	Params map[string]string
}

type Router struct {
	*trie[routerData]
}
