// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.28.0
// 	protoc        v3.12.4
// source: main.proto

package grpc

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type FetchRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *FetchRequest) Reset() {
	*x = FetchRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_main_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *FetchRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*FetchRequest) ProtoMessage() {}

func (x *FetchRequest) ProtoReflect() protoreflect.Message {
	mi := &file_main_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use FetchRequest.ProtoReflect.Descriptor instead.
func (*FetchRequest) Descriptor() ([]byte, []int) {
	return file_main_proto_rawDescGZIP(), []int{0}
}

type FetchMultiRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	N uint32 `protobuf:"varint,1,opt,name=n,proto3" json:"n,omitempty"`
}

func (x *FetchMultiRequest) Reset() {
	*x = FetchMultiRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_main_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *FetchMultiRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*FetchMultiRequest) ProtoMessage() {}

func (x *FetchMultiRequest) ProtoReflect() protoreflect.Message {
	mi := &file_main_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use FetchMultiRequest.ProtoReflect.Descriptor instead.
func (*FetchMultiRequest) Descriptor() ([]byte, []int) {
	return file_main_proto_rawDescGZIP(), []int{1}
}

func (x *FetchMultiRequest) GetN() uint32 {
	if x != nil {
		return x.N
	}
	return 0
}

type FetchResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Id uint64 `protobuf:"varint,1,opt,name=id,proto3" json:"id,omitempty"`
}

func (x *FetchResponse) Reset() {
	*x = FetchResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_main_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *FetchResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*FetchResponse) ProtoMessage() {}

func (x *FetchResponse) ProtoReflect() protoreflect.Message {
	mi := &file_main_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use FetchResponse.ProtoReflect.Descriptor instead.
func (*FetchResponse) Descriptor() ([]byte, []int) {
	return file_main_proto_rawDescGZIP(), []int{2}
}

func (x *FetchResponse) GetId() uint64 {
	if x != nil {
		return x.Id
	}
	return 0
}

type FetchMultiResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Ids []uint64 `protobuf:"varint,1,rep,packed,name=ids,proto3" json:"ids,omitempty"`
}

func (x *FetchMultiResponse) Reset() {
	*x = FetchMultiResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_main_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *FetchMultiResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*FetchMultiResponse) ProtoMessage() {}

func (x *FetchMultiResponse) ProtoReflect() protoreflect.Message {
	mi := &file_main_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use FetchMultiResponse.ProtoReflect.Descriptor instead.
func (*FetchMultiResponse) Descriptor() ([]byte, []int) {
	return file_main_proto_rawDescGZIP(), []int{3}
}

func (x *FetchMultiResponse) GetIds() []uint64 {
	if x != nil {
		return x.Ids
	}
	return nil
}

type StatsRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *StatsRequest) Reset() {
	*x = StatsRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_main_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *StatsRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*StatsRequest) ProtoMessage() {}

func (x *StatsRequest) ProtoReflect() protoreflect.Message {
	mi := &file_main_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use StatsRequest.ProtoReflect.Descriptor instead.
func (*StatsRequest) Descriptor() ([]byte, []int) {
	return file_main_proto_rawDescGZIP(), []int{4}
}

type StatsResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Pid              int32  `protobuf:"varint,1,opt,name=pid,proto3" json:"pid,omitempty"`
	Uptime           int64  `protobuf:"varint,2,opt,name=uptime,proto3" json:"uptime,omitempty"`
	Time             int64  `protobuf:"varint,3,opt,name=time,proto3" json:"time,omitempty"`
	Version          string `protobuf:"bytes,4,opt,name=version,proto3" json:"version,omitempty"`
	CurrConnections  int64  `protobuf:"varint,5,opt,name=curr_connections,json=currConnections,proto3" json:"curr_connections,omitempty"`
	TotalConnections int64  `protobuf:"varint,6,opt,name=total_connections,json=totalConnections,proto3" json:"total_connections,omitempty"`
	CmdGet           int64  `protobuf:"varint,7,opt,name=cmd_get,json=cmdGet,proto3" json:"cmd_get,omitempty"`
	GetHits          int64  `protobuf:"varint,8,opt,name=get_hits,json=getHits,proto3" json:"get_hits,omitempty"`
	GetMisses        int64  `protobuf:"varint,9,opt,name=get_misses,json=getMisses,proto3" json:"get_misses,omitempty"`
}

func (x *StatsResponse) Reset() {
	*x = StatsResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_main_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *StatsResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*StatsResponse) ProtoMessage() {}

func (x *StatsResponse) ProtoReflect() protoreflect.Message {
	mi := &file_main_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use StatsResponse.ProtoReflect.Descriptor instead.
func (*StatsResponse) Descriptor() ([]byte, []int) {
	return file_main_proto_rawDescGZIP(), []int{5}
}

func (x *StatsResponse) GetPid() int32 {
	if x != nil {
		return x.Pid
	}
	return 0
}

func (x *StatsResponse) GetUptime() int64 {
	if x != nil {
		return x.Uptime
	}
	return 0
}

func (x *StatsResponse) GetTime() int64 {
	if x != nil {
		return x.Time
	}
	return 0
}

func (x *StatsResponse) GetVersion() string {
	if x != nil {
		return x.Version
	}
	return ""
}

func (x *StatsResponse) GetCurrConnections() int64 {
	if x != nil {
		return x.CurrConnections
	}
	return 0
}

func (x *StatsResponse) GetTotalConnections() int64 {
	if x != nil {
		return x.TotalConnections
	}
	return 0
}

func (x *StatsResponse) GetCmdGet() int64 {
	if x != nil {
		return x.CmdGet
	}
	return 0
}

func (x *StatsResponse) GetGetHits() int64 {
	if x != nil {
		return x.GetHits
	}
	return 0
}

func (x *StatsResponse) GetGetMisses() int64 {
	if x != nil {
		return x.GetMisses
	}
	return 0
}

var File_main_proto protoreflect.FileDescriptor

var file_main_proto_rawDesc = []byte{
	0x0a, 0x0a, 0x6d, 0x61, 0x69, 0x6e, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x0a, 0x6b, 0x61,
	0x74, 0x73, 0x75, 0x62, 0x75, 0x73, 0x68, 0x69, 0x22, 0x0e, 0x0a, 0x0c, 0x46, 0x65, 0x74, 0x63,
	0x68, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x22, 0x21, 0x0a, 0x11, 0x46, 0x65, 0x74, 0x63,
	0x68, 0x4d, 0x75, 0x6c, 0x74, 0x69, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x0c, 0x0a,
	0x01, 0x6e, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x01, 0x6e, 0x22, 0x1f, 0x0a, 0x0d, 0x46,
	0x65, 0x74, 0x63, 0x68, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x0e, 0x0a, 0x02,
	0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x04, 0x52, 0x02, 0x69, 0x64, 0x22, 0x26, 0x0a, 0x12,
	0x46, 0x65, 0x74, 0x63, 0x68, 0x4d, 0x75, 0x6c, 0x74, 0x69, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e,
	0x73, 0x65, 0x12, 0x10, 0x0a, 0x03, 0x69, 0x64, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x04, 0x52,
	0x03, 0x69, 0x64, 0x73, 0x22, 0x0e, 0x0a, 0x0c, 0x53, 0x74, 0x61, 0x74, 0x73, 0x52, 0x65, 0x71,
	0x75, 0x65, 0x73, 0x74, 0x22, 0x92, 0x02, 0x0a, 0x0d, 0x53, 0x74, 0x61, 0x74, 0x73, 0x52, 0x65,
	0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x10, 0x0a, 0x03, 0x70, 0x69, 0x64, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x05, 0x52, 0x03, 0x70, 0x69, 0x64, 0x12, 0x16, 0x0a, 0x06, 0x75, 0x70, 0x74, 0x69,
	0x6d, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x03, 0x52, 0x06, 0x75, 0x70, 0x74, 0x69, 0x6d, 0x65,
	0x12, 0x12, 0x0a, 0x04, 0x74, 0x69, 0x6d, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28, 0x03, 0x52, 0x04,
	0x74, 0x69, 0x6d, 0x65, 0x12, 0x18, 0x0a, 0x07, 0x76, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e, 0x18,
	0x04, 0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x76, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e, 0x12, 0x29,
	0x0a, 0x10, 0x63, 0x75, 0x72, 0x72, 0x5f, 0x63, 0x6f, 0x6e, 0x6e, 0x65, 0x63, 0x74, 0x69, 0x6f,
	0x6e, 0x73, 0x18, 0x05, 0x20, 0x01, 0x28, 0x03, 0x52, 0x0f, 0x63, 0x75, 0x72, 0x72, 0x43, 0x6f,
	0x6e, 0x6e, 0x65, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x12, 0x2b, 0x0a, 0x11, 0x74, 0x6f, 0x74,
	0x61, 0x6c, 0x5f, 0x63, 0x6f, 0x6e, 0x6e, 0x65, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x18, 0x06,
	0x20, 0x01, 0x28, 0x03, 0x52, 0x10, 0x74, 0x6f, 0x74, 0x61, 0x6c, 0x43, 0x6f, 0x6e, 0x6e, 0x65,
	0x63, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x12, 0x17, 0x0a, 0x07, 0x63, 0x6d, 0x64, 0x5f, 0x67, 0x65,
	0x74, 0x18, 0x07, 0x20, 0x01, 0x28, 0x03, 0x52, 0x06, 0x63, 0x6d, 0x64, 0x47, 0x65, 0x74, 0x12,
	0x19, 0x0a, 0x08, 0x67, 0x65, 0x74, 0x5f, 0x68, 0x69, 0x74, 0x73, 0x18, 0x08, 0x20, 0x01, 0x28,
	0x03, 0x52, 0x07, 0x67, 0x65, 0x74, 0x48, 0x69, 0x74, 0x73, 0x12, 0x1d, 0x0a, 0x0a, 0x67, 0x65,
	0x74, 0x5f, 0x6d, 0x69, 0x73, 0x73, 0x65, 0x73, 0x18, 0x09, 0x20, 0x01, 0x28, 0x03, 0x52, 0x09,
	0x67, 0x65, 0x74, 0x4d, 0x69, 0x73, 0x73, 0x65, 0x73, 0x32, 0x9a, 0x01, 0x0a, 0x09, 0x47, 0x65,
	0x6e, 0x65, 0x72, 0x61, 0x74, 0x6f, 0x72, 0x12, 0x3e, 0x0a, 0x05, 0x46, 0x65, 0x74, 0x63, 0x68,
	0x12, 0x18, 0x2e, 0x6b, 0x61, 0x74, 0x73, 0x75, 0x62, 0x75, 0x73, 0x68, 0x69, 0x2e, 0x46, 0x65,
	0x74, 0x63, 0x68, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x19, 0x2e, 0x6b, 0x61, 0x74,
	0x73, 0x75, 0x62, 0x75, 0x73, 0x68, 0x69, 0x2e, 0x46, 0x65, 0x74, 0x63, 0x68, 0x52, 0x65, 0x73,
	0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x00, 0x12, 0x4d, 0x0a, 0x0a, 0x46, 0x65, 0x74, 0x63, 0x68,
	0x4d, 0x75, 0x6c, 0x74, 0x69, 0x12, 0x1d, 0x2e, 0x6b, 0x61, 0x74, 0x73, 0x75, 0x62, 0x75, 0x73,
	0x68, 0x69, 0x2e, 0x46, 0x65, 0x74, 0x63, 0x68, 0x4d, 0x75, 0x6c, 0x74, 0x69, 0x52, 0x65, 0x71,
	0x75, 0x65, 0x73, 0x74, 0x1a, 0x1e, 0x2e, 0x6b, 0x61, 0x74, 0x73, 0x75, 0x62, 0x75, 0x73, 0x68,
	0x69, 0x2e, 0x46, 0x65, 0x74, 0x63, 0x68, 0x4d, 0x75, 0x6c, 0x74, 0x69, 0x52, 0x65, 0x73, 0x70,
	0x6f, 0x6e, 0x73, 0x65, 0x22, 0x00, 0x32, 0x45, 0x0a, 0x05, 0x53, 0x74, 0x61, 0x74, 0x73, 0x12,
	0x3c, 0x0a, 0x03, 0x47, 0x65, 0x74, 0x12, 0x18, 0x2e, 0x6b, 0x61, 0x74, 0x73, 0x75, 0x62, 0x75,
	0x73, 0x68, 0x69, 0x2e, 0x53, 0x74, 0x61, 0x74, 0x73, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74,
	0x1a, 0x19, 0x2e, 0x6b, 0x61, 0x74, 0x73, 0x75, 0x62, 0x75, 0x73, 0x68, 0x69, 0x2e, 0x53, 0x74,
	0x61, 0x74, 0x73, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x00, 0x42, 0x11, 0x5a,
	0x0f, 0x6b, 0x61, 0x74, 0x73, 0x75, 0x62, 0x75, 0x73, 0x68, 0x69, 0x2f, 0x67, 0x72, 0x70, 0x63,
	0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_main_proto_rawDescOnce sync.Once
	file_main_proto_rawDescData = file_main_proto_rawDesc
)

func file_main_proto_rawDescGZIP() []byte {
	file_main_proto_rawDescOnce.Do(func() {
		file_main_proto_rawDescData = protoimpl.X.CompressGZIP(file_main_proto_rawDescData)
	})
	return file_main_proto_rawDescData
}

var file_main_proto_msgTypes = make([]protoimpl.MessageInfo, 6)
var file_main_proto_goTypes = []interface{}{
	(*FetchRequest)(nil),       // 0: katsubushi.FetchRequest
	(*FetchMultiRequest)(nil),  // 1: katsubushi.FetchMultiRequest
	(*FetchResponse)(nil),      // 2: katsubushi.FetchResponse
	(*FetchMultiResponse)(nil), // 3: katsubushi.FetchMultiResponse
	(*StatsRequest)(nil),       // 4: katsubushi.StatsRequest
	(*StatsResponse)(nil),      // 5: katsubushi.StatsResponse
}
var file_main_proto_depIdxs = []int32{
	0, // 0: katsubushi.Generator.Fetch:input_type -> katsubushi.FetchRequest
	1, // 1: katsubushi.Generator.FetchMulti:input_type -> katsubushi.FetchMultiRequest
	4, // 2: katsubushi.Stats.Get:input_type -> katsubushi.StatsRequest
	2, // 3: katsubushi.Generator.Fetch:output_type -> katsubushi.FetchResponse
	3, // 4: katsubushi.Generator.FetchMulti:output_type -> katsubushi.FetchMultiResponse
	5, // 5: katsubushi.Stats.Get:output_type -> katsubushi.StatsResponse
	3, // [3:6] is the sub-list for method output_type
	0, // [0:3] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_main_proto_init() }
func file_main_proto_init() {
	if File_main_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_main_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*FetchRequest); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_main_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*FetchMultiRequest); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_main_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*FetchResponse); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_main_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*FetchMultiResponse); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_main_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*StatsRequest); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_main_proto_msgTypes[5].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*StatsResponse); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_main_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   6,
			NumExtensions: 0,
			NumServices:   2,
		},
		GoTypes:           file_main_proto_goTypes,
		DependencyIndexes: file_main_proto_depIdxs,
		MessageInfos:      file_main_proto_msgTypes,
	}.Build()
	File_main_proto = out.File
	file_main_proto_rawDesc = nil
	file_main_proto_goTypes = nil
	file_main_proto_depIdxs = nil
}
