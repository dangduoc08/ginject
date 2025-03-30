package devtool

type RESTRequest struct {
	Body   []Schema `json:"body"`
	Form   []Schema `json:"form"`
	Query  []Schema `json:"query"`
	Header []Schema `json:"header"`
	Param  []Schema `json:"param"`
	File   []Schema `json:"file"`
}

type RESTVersioning struct {
	Type  int    `json:"type"`
	Value string `json:"value"`
	Key   string `json:"key"`
}

type RESTComponent struct {
	// ID         string         `json:"id"`
	Handler    string         `json:"handler"`
	HTTPMethod string         `json:"http_method"`
	Route      string         `json:"route"`
	Versioning RESTVersioning `json:"versioning"`
	Request    RESTRequest    `json:"request"`
}
