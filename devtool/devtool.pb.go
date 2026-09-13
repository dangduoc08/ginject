package devtool

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
	unsafe "unsafe"
)

const (
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)

	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type LayerScope int32

const (
	LayerScope_UNKNOWN       LayerScope = 0
	LayerScope_REQUEST_SCOPE LayerScope = 1
	LayerScope_GLOBAL_SCOPE  LayerScope = 2
)

var (
	LayerScope_name = map[int32]string{
		0: "UNKNOWN",
		1: "REQUEST_SCOPE",
		2: "GLOBAL_SCOPE",
	}
	LayerScope_value = map[string]int32{
		"UNKNOWN":       0,
		"REQUEST_SCOPE": 1,
		"GLOBAL_SCOPE":  2,
	}
)

func (x LayerScope) Enum() *LayerScope {
	p := new(LayerScope)
	*p = x
	return p
}

func (x LayerScope) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (LayerScope) Descriptor() protoreflect.EnumDescriptor {
	return file_devtool_devtool_proto_enumTypes[0].Descriptor()
}

func (LayerScope) Type() protoreflect.EnumType {
	return &file_devtool_devtool_proto_enumTypes[0]
}

func (x LayerScope) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

func (LayerScope) EnumDescriptor() ([]byte, []int) {
	return file_devtool_devtool_proto_rawDescGZIP(), []int{0}
}

type GetConfigurationRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *GetConfigurationRequest) Reset() {
	*x = GetConfigurationRequest{}
	mi := &file_devtool_devtool_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *GetConfigurationRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetConfigurationRequest) ProtoMessage() {}

func (x *GetConfigurationRequest) ProtoReflect() protoreflect.Message {
	mi := &file_devtool_devtool_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (*GetConfigurationRequest) Descriptor() ([]byte, []int) {
	return file_devtool_devtool_proto_rawDescGZIP(), []int{0}
}

type GetConfigurationResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Controller    *Controller            `protobuf:"bytes,1,opt,name=controller,proto3" json:"controller,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *GetConfigurationResponse) Reset() {
	*x = GetConfigurationResponse{}
	mi := &file_devtool_devtool_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *GetConfigurationResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetConfigurationResponse) ProtoMessage() {}

func (x *GetConfigurationResponse) ProtoReflect() protoreflect.Message {
	mi := &file_devtool_devtool_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (*GetConfigurationResponse) Descriptor() ([]byte, []int) {
	return file_devtool_devtool_proto_rawDescGZIP(), []int{1}
}

func (x *GetConfigurationResponse) GetController() *Controller {
	if x != nil {
		return x.Controller
	}
	return nil
}

type Controller struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Http          []*HTTPComponent       `protobuf:"bytes,1,rep,name=http,proto3" json:"http,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Controller) Reset() {
	*x = Controller{}
	mi := &file_devtool_devtool_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Controller) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Controller) ProtoMessage() {}

func (x *Controller) ProtoReflect() protoreflect.Message {
	mi := &file_devtool_devtool_proto_msgTypes[2]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (*Controller) Descriptor() ([]byte, []int) {
	return file_devtool_devtool_proto_rawDescGZIP(), []int{2}
}

func (x *Controller) GetHttp() []*HTTPComponent {
	if x != nil {
		return x.Http
	}
	return nil
}

type HTTPComponent struct {
	state            protoimpl.MessageState `protogen:"open.v1"`
	Id               string                 `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Handler          string                 `protobuf:"bytes,2,opt,name=handler,proto3" json:"handler,omitempty"`
	HttpMethod       string                 `protobuf:"bytes,3,opt,name=http_method,json=httpMethod,proto3" json:"http_method,omitempty"`
	Route            string                 `protobuf:"bytes,4,opt,name=route,proto3" json:"route,omitempty"`
	ExceptionFilters []*Layer               `protobuf:"bytes,5,rep,name=exception_filters,json=exceptionFilters,proto3" json:"exception_filters,omitempty"`
	Middlewares      []*Layer               `protobuf:"bytes,6,rep,name=middlewares,proto3" json:"middlewares,omitempty"`
	Guards           []*Layer               `protobuf:"bytes,7,rep,name=guards,proto3" json:"guards,omitempty"`
	Interceptors     []*Layer               `protobuf:"bytes,8,rep,name=interceptors,proto3" json:"interceptors,omitempty"`
	Versioning       *HTTPVersioning        `protobuf:"bytes,9,opt,name=versioning,proto3" json:"versioning,omitempty"`
	Request          *HTTPRequest           `protobuf:"bytes,10,opt,name=request,proto3" json:"request,omitempty"`
	unknownFields    protoimpl.UnknownFields
	sizeCache        protoimpl.SizeCache
}

func (x *HTTPComponent) Reset() {
	*x = HTTPComponent{}
	mi := &file_devtool_devtool_proto_msgTypes[3]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *HTTPComponent) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*HTTPComponent) ProtoMessage() {}

func (x *HTTPComponent) ProtoReflect() protoreflect.Message {
	mi := &file_devtool_devtool_proto_msgTypes[3]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (*HTTPComponent) Descriptor() ([]byte, []int) {
	return file_devtool_devtool_proto_rawDescGZIP(), []int{3}
}

func (x *HTTPComponent) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *HTTPComponent) GetHandler() string {
	if x != nil {
		return x.Handler
	}
	return ""
}

func (x *HTTPComponent) GetHttpMethod() string {
	if x != nil {
		return x.HttpMethod
	}
	return ""
}

func (x *HTTPComponent) GetRoute() string {
	if x != nil {
		return x.Route
	}
	return ""
}

func (x *HTTPComponent) GetExceptionFilters() []*Layer {
	if x != nil {
		return x.ExceptionFilters
	}
	return nil
}

func (x *HTTPComponent) GetMiddlewares() []*Layer {
	if x != nil {
		return x.Middlewares
	}
	return nil
}

func (x *HTTPComponent) GetGuards() []*Layer {
	if x != nil {
		return x.Guards
	}
	return nil
}

func (x *HTTPComponent) GetInterceptors() []*Layer {
	if x != nil {
		return x.Interceptors
	}
	return nil
}

func (x *HTTPComponent) GetVersioning() *HTTPVersioning {
	if x != nil {
		return x.Versioning
	}
	return nil
}

func (x *HTTPComponent) GetRequest() *HTTPRequest {
	if x != nil {
		return x.Request
	}
	return nil
}

type Layer struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Scope         LayerScope             `protobuf:"varint,1,opt,name=scope,proto3,enum=devtool.LayerScope" json:"scope,omitempty"`
	Name          string                 `protobuf:"bytes,2,opt,name=name,proto3" json:"name,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Layer) Reset() {
	*x = Layer{}
	mi := &file_devtool_devtool_proto_msgTypes[4]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Layer) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Layer) ProtoMessage() {}

func (x *Layer) ProtoReflect() protoreflect.Message {
	mi := &file_devtool_devtool_proto_msgTypes[4]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (*Layer) Descriptor() ([]byte, []int) {
	return file_devtool_devtool_proto_rawDescGZIP(), []int{4}
}

func (x *Layer) GetScope() LayerScope {
	if x != nil {
		return x.Scope
	}
	return LayerScope_UNKNOWN
}

func (x *Layer) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

type HTTPVersioning struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Type          int32                  `protobuf:"varint,1,opt,name=type,proto3" json:"type,omitempty"`
	Value         string                 `protobuf:"bytes,2,opt,name=value,proto3" json:"value,omitempty"`
	Key           string                 `protobuf:"bytes,3,opt,name=key,proto3" json:"key,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *HTTPVersioning) Reset() {
	*x = HTTPVersioning{}
	mi := &file_devtool_devtool_proto_msgTypes[5]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *HTTPVersioning) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*HTTPVersioning) ProtoMessage() {}

func (x *HTTPVersioning) ProtoReflect() protoreflect.Message {
	mi := &file_devtool_devtool_proto_msgTypes[5]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (*HTTPVersioning) Descriptor() ([]byte, []int) {
	return file_devtool_devtool_proto_rawDescGZIP(), []int{5}
}

func (x *HTTPVersioning) GetType() int32 {
	if x != nil {
		return x.Type
	}
	return 0
}

func (x *HTTPVersioning) GetValue() string {
	if x != nil {
		return x.Value
	}
	return ""
}

func (x *HTTPVersioning) GetKey() string {
	if x != nil {
		return x.Key
	}
	return ""
}

type HTTPRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Body          []*Schema              `protobuf:"bytes,1,rep,name=body,proto3" json:"body,omitempty"`
	Form          []*Schema              `protobuf:"bytes,2,rep,name=form,proto3" json:"form,omitempty"`
	Query         []*Schema              `protobuf:"bytes,3,rep,name=query,proto3" json:"query,omitempty"`
	Header        []*Schema              `protobuf:"bytes,4,rep,name=header,proto3" json:"header,omitempty"`
	Param         []*Schema              `protobuf:"bytes,5,rep,name=param,proto3" json:"param,omitempty"`
	File          []*Schema              `protobuf:"bytes,6,rep,name=file,proto3" json:"file,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *HTTPRequest) Reset() {
	*x = HTTPRequest{}
	mi := &file_devtool_devtool_proto_msgTypes[6]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *HTTPRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*HTTPRequest) ProtoMessage() {}

func (x *HTTPRequest) ProtoReflect() protoreflect.Message {
	mi := &file_devtool_devtool_proto_msgTypes[6]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (*HTTPRequest) Descriptor() ([]byte, []int) {
	return file_devtool_devtool_proto_rawDescGZIP(), []int{6}
}

func (x *HTTPRequest) GetBody() []*Schema {
	if x != nil {
		return x.Body
	}
	return nil
}

func (x *HTTPRequest) GetForm() []*Schema {
	if x != nil {
		return x.Form
	}
	return nil
}

func (x *HTTPRequest) GetQuery() []*Schema {
	if x != nil {
		return x.Query
	}
	return nil
}

func (x *HTTPRequest) GetHeader() []*Schema {
	if x != nil {
		return x.Header
	}
	return nil
}

func (x *HTTPRequest) GetParam() []*Schema {
	if x != nil {
		return x.Param
	}
	return nil
}

func (x *HTTPRequest) GetFile() []*Schema {
	if x != nil {
		return x.File
	}
	return nil
}

type Schema struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Name          string                 `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	Type          string                 `protobuf:"bytes,2,opt,name=type,proto3" json:"type,omitempty"`
	Format        string                 `protobuf:"bytes,3,opt,name=format,proto3" json:"format,omitempty"`
	Item          *Schema                `protobuf:"bytes,4,opt,name=item,proto3" json:"item,omitempty"`
	Properties    []*Schema              `protobuf:"bytes,5,rep,name=properties,proto3" json:"properties,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Schema) Reset() {
	*x = Schema{}
	mi := &file_devtool_devtool_proto_msgTypes[7]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Schema) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Schema) ProtoMessage() {}

func (x *Schema) ProtoReflect() protoreflect.Message {
	mi := &file_devtool_devtool_proto_msgTypes[7]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (*Schema) Descriptor() ([]byte, []int) {
	return file_devtool_devtool_proto_rawDescGZIP(), []int{7}
}

func (x *Schema) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *Schema) GetType() string {
	if x != nil {
		return x.Type
	}
	return ""
}

func (x *Schema) GetFormat() string {
	if x != nil {
		return x.Format
	}
	return ""
}

func (x *Schema) GetItem() *Schema {
	if x != nil {
		return x.Item
	}
	return nil
}

func (x *Schema) GetProperties() []*Schema {
	if x != nil {
		return x.Properties
	}
	return nil
}

var File_devtool_devtool_proto protoreflect.FileDescriptor

const file_devtool_devtool_proto_rawDesc = "" +
	"\n" +
	"\x15devtool/devtool.proto\x12\adevtool\"\x19\n" +
	"\x17GetConfigurationRequest\"O\n" +
	"\x18GetConfigurationResponse\x123\n" +
	"\n" +
	"controller\x18\x01 \x01(\v2\x13.devtool.ControllerR\n" +
	"controller\"8\n" +
	"\n" +
	"Controller\x12*\n" +
	"\x04http\x18\x01 \x03(\v2\x16.devtool.HTTPComponentR\x04http\"\xa4\x03\n" +
	"\rHTTPComponent\x12\x0e\n" +
	"\x02id\x18\x01 \x01(\tR\x02id\x12\x18\n" +
	"\ahandler\x18\x02 \x01(\tR\ahandler\x12\x1f\n" +
	"\vhttp_method\x18\x03 \x01(\tR\n" +
	"httpMethod\x12\x14\n" +
	"\x05route\x18\x04 \x01(\tR\x05route\x12;\n" +
	"\x11exception_filters\x18\x05 \x03(\v2\x0e.devtool.LayerR\x10exceptionFilters\x120\n" +
	"\vmiddlewares\x18\x06 \x03(\v2\x0e.devtool.LayerR\vmiddlewares\x12&\n" +
	"\x06guards\x18\a \x03(\v2\x0e.devtool.LayerR\x06guards\x122\n" +
	"\finterceptors\x18\b \x03(\v2\x0e.devtool.LayerR\finterceptors\x127\n" +
	"\n" +
	"versioning\x18\t \x01(\v2\x17.devtool.HTTPVersioningR\n" +
	"versioning\x12.\n" +
	"\arequest\x18\n" +
	" \x01(\v2\x14.devtool.HTTPRequestR\arequest\"F\n" +
	"\x05Layer\x12)\n" +
	"\x05scope\x18\x01 \x01(\x0e2\x13.devtool.LayerScopeR\x05scope\x12\x12\n" +
	"\x04name\x18\x02 \x01(\tR\x04name\"L\n" +
	"\x0eHTTPVersioning\x12\x12\n" +
	"\x04type\x18\x01 \x01(\x05R\x04type\x12\x14\n" +
	"\x05value\x18\x02 \x01(\tR\x05value\x12\x10\n" +
	"\x03key\x18\x03 \x01(\tR\x03key\"\xf3\x01\n" +
	"\vHTTPRequest\x12#\n" +
	"\x04body\x18\x01 \x03(\v2\x0f.devtool.SchemaR\x04body\x12#\n" +
	"\x04form\x18\x02 \x03(\v2\x0f.devtool.SchemaR\x04form\x12%\n" +
	"\x05query\x18\x03 \x03(\v2\x0f.devtool.SchemaR\x05query\x12'\n" +
	"\x06header\x18\x04 \x03(\v2\x0f.devtool.SchemaR\x06header\x12%\n" +
	"\x05param\x18\x05 \x03(\v2\x0f.devtool.SchemaR\x05param\x12#\n" +
	"\x04file\x18\x06 \x03(\v2\x0f.devtool.SchemaR\x04file\"\x9e\x01\n" +
	"\x06Schema\x12\x12\n" +
	"\x04name\x18\x01 \x01(\tR\x04name\x12\x12\n" +
	"\x04type\x18\x02 \x01(\tR\x04type\x12\x16\n" +
	"\x06format\x18\x03 \x01(\tR\x06format\x12#\n" +
	"\x04item\x18\x04 \x01(\v2\x0f.devtool.SchemaR\x04item\x12/\n" +
	"\n" +
	"properties\x18\x05 \x03(\v2\x0f.devtool.SchemaR\n" +
	"properties*>\n" +
	"\n" +
	"LayerScope\x12\v\n" +
	"\aUNKNOWN\x10\x00\x12\x11\n" +
	"\rREQUEST_SCOPE\x10\x01\x12\x10\n" +
	"\fGLOBAL_SCOPE\x10\x022k\n" +
	"\x0eDevtoolService\x12Y\n" +
	"\x10GetConfiguration\x12 .devtool.GetConfigurationRequest\x1a!.devtool.GetConfigurationResponse\"\x00B/Z-github.com/dangduoc08/ginject/devtool/devtoolb\x06proto3"

var (
	file_devtool_devtool_proto_rawDescOnce sync.Once
	file_devtool_devtool_proto_rawDescData []byte
)

func file_devtool_devtool_proto_rawDescGZIP() []byte {
	file_devtool_devtool_proto_rawDescOnce.Do(func() {
		file_devtool_devtool_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_devtool_devtool_proto_rawDesc), len(file_devtool_devtool_proto_rawDesc)))
	})
	return file_devtool_devtool_proto_rawDescData
}

var file_devtool_devtool_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_devtool_devtool_proto_msgTypes = make([]protoimpl.MessageInfo, 8)
var file_devtool_devtool_proto_goTypes = []any{
	(LayerScope)(0),
	(*GetConfigurationRequest)(nil),
	(*GetConfigurationResponse)(nil),
	(*Controller)(nil),
	(*HTTPComponent)(nil),
	(*Layer)(nil),
	(*HTTPVersioning)(nil),
	(*HTTPRequest)(nil),
	(*Schema)(nil),
}
var file_devtool_devtool_proto_depIdxs = []int32{
	3,
	4,
	5,
	5,
	5,
	5,
	6,
	7,
	0,
	8,
	8,
	8,
	8,
	8,
	8,
	8,
	8,
	1,
	2,
	18,
	17,
	17,
	17,
	0,
}

func init() { file_devtool_devtool_proto_init() }
func file_devtool_devtool_proto_init() {
	if File_devtool_devtool_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_devtool_devtool_proto_rawDesc), len(file_devtool_devtool_proto_rawDesc)),
			NumEnums:      1,
			NumMessages:   8,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_devtool_devtool_proto_goTypes,
		DependencyIndexes: file_devtool_devtool_proto_depIdxs,
		EnumInfos:         file_devtool_devtool_proto_enumTypes,
		MessageInfos:      file_devtool_devtool_proto_msgTypes,
	}.Build()
	File_devtool_devtool_proto = out.File
	file_devtool_devtool_proto_goTypes = nil
	file_devtool_devtool_proto_depIdxs = nil
}
