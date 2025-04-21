package devtool

type LayerScope string

const (
	REQUEST_SCOPE LayerScope = "request"
	GLOBAL_SCOPE  LayerScope = "global"
)

type RESTLayer struct {
	Scope LayerScope `json:"scope"`
	Name  string     `json:"name"`
}

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
	ID               string         `json:"id"`
	Handler          string         `json:"handler"`
	HTTPMethod       string         `json:"http_method"`
	Route            string         `json:"route"`
	ExceptionFilters []RESTLayer    `json:"exception_filters"`
	Middlewares      []RESTLayer    `json:"middlewares"`
	Guards           []RESTLayer    `json:"guards"`
	Interceptors     []RESTLayer    `json:"interceptors"`
	Versioning       RESTVersioning `json:"versioning"`
	Request          RESTRequest    `json:"request"`
}

type DevtoolController struct {
	REST []RESTComponent `json:"rest"`
}

type Devtool struct {
	Controller DevtoolController `json:"controllers"`
}

func (d *Devtool) Serve() {

	// tls := flag.Bool("tls", false, "Connection uses TLS if true, else plain TCP")

	// fmt.Println(tls)

	// listener, err := net.Listen("tcp", "localhost:50051")
	// fmt.Println("server listen on: localhost:50051")

	// if err != nil {
	// 	log.Fatalf("error %v", err)
	// }

	// server := grpc.NewServer(grpc.)

	// calculatorpb.RegisterCalculatorServiceServer(server, &Server{})

	// if err := server.Serve(listener); err != nil {
	// 	log.Fatalf("failed to listen: %v", err)
	// }

	// client := NewGreeterClient(conn)

	// // Gửi request đến server
	// ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	// defer cancel()

	// res, err := client.SayHello(ctx, &HelloRequest{Name: "Alice"})
	// if err != nil {
	// 	log.Fatalf("Error calling SayHello: %v", err)
	// }

	// fmt.Println("Response from server:", res.Message)
}
