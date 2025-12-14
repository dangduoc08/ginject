package devtool

import (
	context "context"
)

type Devtool struct {
	GetConfigurationResponse
	DevtoolServiceServer
}

func (devtool *Devtool) GetConfiguration(context.Context, *GetConfigurationRequest) (*GetConfigurationResponse, error) {

	return &GetConfigurationResponse{
		Controller: devtool.Controller,
	}, nil
}

func (devtool *Devtool) Serve() {
	// lis, err := net.Listen("tcp", ":50051")
	// if err != nil {
	// 	log.Fatalf("failed to listen: %v", err)
	// }

	// s := grpc.NewServer()

	// RegisterDevtoolServiceServer(s, devtool)

	// fmt.Println("Server is running on port 50051...")
	// if err := s.Serve(lis); err != nil {
	// 	log.Fatalf("failed to serve: %v", err)
	// }

}
