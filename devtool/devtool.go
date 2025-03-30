package devtool

type DevtoolController struct {
	REST []RESTComponent `json:"rest"`
}

type DevtoolBuilder struct {
	Controller DevtoolController
}

type Devtool struct {
	Controller DevtoolController `json:"controllers"`
}

func NewDevtoolBuilder() *DevtoolBuilder {
	return &DevtoolBuilder{}
}

func (builder *DevtoolBuilder) AddREST(controllerPath string, restComponent RESTComponent) *DevtoolBuilder {
	// restComponent.ID = d.generateHandlerID(controllerPath + restComponent.Handler)
	builder.Controller.REST = append(builder.Controller.REST, restComponent)
	return builder
}

func (builder *DevtoolBuilder) Build() *Devtool {
	return &Devtool{
		Controller: builder.Controller,
	}
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
