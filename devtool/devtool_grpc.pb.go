package devtool

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

const _ = grpc.SupportPackageIsVersion9

const (
	DevtoolService_GetConfiguration_FullMethodName = "/devtool.DevtoolService/GetConfiguration"
)

type DevtoolServiceClient interface {
	GetConfiguration(ctx context.Context, in *GetConfigurationRequest, opts ...grpc.CallOption) (*GetConfigurationResponse, error)
}

type devtoolServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewDevtoolServiceClient(cc grpc.ClientConnInterface) DevtoolServiceClient {
	return &devtoolServiceClient{cc}
}

func (c *devtoolServiceClient) GetConfiguration(ctx context.Context, in *GetConfigurationRequest, opts ...grpc.CallOption) (*GetConfigurationResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(GetConfigurationResponse)
	err := c.cc.Invoke(ctx, DevtoolService_GetConfiguration_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

type DevtoolServiceServer interface {
	GetConfiguration(context.Context, *GetConfigurationRequest) (*GetConfigurationResponse, error)
	mustEmbedUnimplementedDevtoolServiceServer()
}

type UnimplementedDevtoolServiceServer struct{}

func (UnimplementedDevtoolServiceServer) GetConfiguration(context.Context, *GetConfigurationRequest) (*GetConfigurationResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetConfiguration not implemented")
}
func (UnimplementedDevtoolServiceServer) mustEmbedUnimplementedDevtoolServiceServer() {}
func (UnimplementedDevtoolServiceServer) testEmbeddedByValue()                        {}

type UnsafeDevtoolServiceServer interface {
	mustEmbedUnimplementedDevtoolServiceServer()
}

func RegisterDevtoolServiceServer(s grpc.ServiceRegistrar, srv DevtoolServiceServer) {

	if t, ok := srv.(interface{ testEmbeddedByValue() }); ok {
		t.testEmbeddedByValue()
	}
	s.RegisterService(&DevtoolService_ServiceDesc, srv)
}

func _DevtoolService_GetConfiguration_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetConfigurationRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DevtoolServiceServer).GetConfiguration(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: DevtoolService_GetConfiguration_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DevtoolServiceServer).GetConfiguration(ctx, req.(*GetConfigurationRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var DevtoolService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "devtool.DevtoolService",
	HandlerType: (*DevtoolServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "GetConfiguration",
			Handler:    _DevtoolService_GetConfiguration_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "devtool/devtool.proto",
}
