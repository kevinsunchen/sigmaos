// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.31.0
// 	protoc        v4.24.4
// source: hotel/proto/reserve.proto

package proto

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	proto "sigmaos/tracing/proto"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type ReserveRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	CustomerName      string                   `protobuf:"bytes,1,opt,name=customerName,proto3" json:"customerName,omitempty"`
	HotelId           []string                 `protobuf:"bytes,2,rep,name=hotelId,proto3" json:"hotelId,omitempty"`
	InDate            string                   `protobuf:"bytes,3,opt,name=inDate,proto3" json:"inDate,omitempty"`
	OutDate           string                   `protobuf:"bytes,4,opt,name=outDate,proto3" json:"outDate,omitempty"`
	Number            int32                    `protobuf:"varint,5,opt,name=number,proto3" json:"number,omitempty"`
	SpanContextConfig *proto.SpanContextConfig `protobuf:"bytes,6,opt,name=spanContextConfig,proto3" json:"spanContextConfig,omitempty"`
}

func (x *ReserveRequest) Reset() {
	*x = ReserveRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_hotel_proto_reserve_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ReserveRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ReserveRequest) ProtoMessage() {}

func (x *ReserveRequest) ProtoReflect() protoreflect.Message {
	mi := &file_hotel_proto_reserve_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ReserveRequest.ProtoReflect.Descriptor instead.
func (*ReserveRequest) Descriptor() ([]byte, []int) {
	return file_hotel_proto_reserve_proto_rawDescGZIP(), []int{0}
}

func (x *ReserveRequest) GetCustomerName() string {
	if x != nil {
		return x.CustomerName
	}
	return ""
}

func (x *ReserveRequest) GetHotelId() []string {
	if x != nil {
		return x.HotelId
	}
	return nil
}

func (x *ReserveRequest) GetInDate() string {
	if x != nil {
		return x.InDate
	}
	return ""
}

func (x *ReserveRequest) GetOutDate() string {
	if x != nil {
		return x.OutDate
	}
	return ""
}

func (x *ReserveRequest) GetNumber() int32 {
	if x != nil {
		return x.Number
	}
	return 0
}

func (x *ReserveRequest) GetSpanContextConfig() *proto.SpanContextConfig {
	if x != nil {
		return x.SpanContextConfig
	}
	return nil
}

type ReserveResult struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	HotelIds []string `protobuf:"bytes,1,rep,name=hotelIds,proto3" json:"hotelIds,omitempty"`
}

func (x *ReserveResult) Reset() {
	*x = ReserveResult{}
	if protoimpl.UnsafeEnabled {
		mi := &file_hotel_proto_reserve_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ReserveResult) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ReserveResult) ProtoMessage() {}

func (x *ReserveResult) ProtoReflect() protoreflect.Message {
	mi := &file_hotel_proto_reserve_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ReserveResult.ProtoReflect.Descriptor instead.
func (*ReserveResult) Descriptor() ([]byte, []int) {
	return file_hotel_proto_reserve_proto_rawDescGZIP(), []int{1}
}

func (x *ReserveResult) GetHotelIds() []string {
	if x != nil {
		return x.HotelIds
	}
	return nil
}

var File_hotel_proto_reserve_proto protoreflect.FileDescriptor

var file_hotel_proto_reserve_proto_rawDesc = []byte{
	0x0a, 0x19, 0x68, 0x6f, 0x74, 0x65, 0x6c, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x72, 0x65,
	0x73, 0x65, 0x72, 0x76, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x1b, 0x74, 0x72, 0x61,
	0x63, 0x69, 0x6e, 0x67, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x74, 0x72, 0x61, 0x63, 0x69,
	0x6e, 0x67, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0xda, 0x01, 0x0a, 0x0e, 0x52, 0x65, 0x73,
	0x65, 0x72, 0x76, 0x65, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x22, 0x0a, 0x0c, 0x63,
	0x75, 0x73, 0x74, 0x6f, 0x6d, 0x65, 0x72, 0x4e, 0x61, 0x6d, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x0c, 0x63, 0x75, 0x73, 0x74, 0x6f, 0x6d, 0x65, 0x72, 0x4e, 0x61, 0x6d, 0x65, 0x12,
	0x18, 0x0a, 0x07, 0x68, 0x6f, 0x74, 0x65, 0x6c, 0x49, 0x64, 0x18, 0x02, 0x20, 0x03, 0x28, 0x09,
	0x52, 0x07, 0x68, 0x6f, 0x74, 0x65, 0x6c, 0x49, 0x64, 0x12, 0x16, 0x0a, 0x06, 0x69, 0x6e, 0x44,
	0x61, 0x74, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x69, 0x6e, 0x44, 0x61, 0x74,
	0x65, 0x12, 0x18, 0x0a, 0x07, 0x6f, 0x75, 0x74, 0x44, 0x61, 0x74, 0x65, 0x18, 0x04, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x07, 0x6f, 0x75, 0x74, 0x44, 0x61, 0x74, 0x65, 0x12, 0x16, 0x0a, 0x06, 0x6e,
	0x75, 0x6d, 0x62, 0x65, 0x72, 0x18, 0x05, 0x20, 0x01, 0x28, 0x05, 0x52, 0x06, 0x6e, 0x75, 0x6d,
	0x62, 0x65, 0x72, 0x12, 0x40, 0x0a, 0x11, 0x73, 0x70, 0x61, 0x6e, 0x43, 0x6f, 0x6e, 0x74, 0x65,
	0x78, 0x74, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x18, 0x06, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x12,
	0x2e, 0x53, 0x70, 0x61, 0x6e, 0x43, 0x6f, 0x6e, 0x74, 0x65, 0x78, 0x74, 0x43, 0x6f, 0x6e, 0x66,
	0x69, 0x67, 0x52, 0x11, 0x73, 0x70, 0x61, 0x6e, 0x43, 0x6f, 0x6e, 0x74, 0x65, 0x78, 0x74, 0x43,
	0x6f, 0x6e, 0x66, 0x69, 0x67, 0x22, 0x2b, 0x0a, 0x0d, 0x52, 0x65, 0x73, 0x65, 0x72, 0x76, 0x65,
	0x52, 0x65, 0x73, 0x75, 0x6c, 0x74, 0x12, 0x1a, 0x0a, 0x08, 0x68, 0x6f, 0x74, 0x65, 0x6c, 0x49,
	0x64, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x09, 0x52, 0x08, 0x68, 0x6f, 0x74, 0x65, 0x6c, 0x49,
	0x64, 0x73, 0x32, 0x73, 0x0a, 0x07, 0x52, 0x65, 0x73, 0x65, 0x72, 0x76, 0x65, 0x12, 0x32, 0x0a,
	0x0f, 0x4d, 0x61, 0x6b, 0x65, 0x52, 0x65, 0x73, 0x65, 0x72, 0x76, 0x61, 0x74, 0x69, 0x6f, 0x6e,
	0x12, 0x0f, 0x2e, 0x52, 0x65, 0x73, 0x65, 0x72, 0x76, 0x65, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73,
	0x74, 0x1a, 0x0e, 0x2e, 0x52, 0x65, 0x73, 0x65, 0x72, 0x76, 0x65, 0x52, 0x65, 0x73, 0x75, 0x6c,
	0x74, 0x12, 0x34, 0x0a, 0x11, 0x43, 0x68, 0x65, 0x63, 0x6b, 0x41, 0x76, 0x61, 0x69, 0x6c, 0x61,
	0x62, 0x69, 0x6c, 0x69, 0x74, 0x79, 0x12, 0x0f, 0x2e, 0x52, 0x65, 0x73, 0x65, 0x72, 0x76, 0x65,
	0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x0e, 0x2e, 0x52, 0x65, 0x73, 0x65, 0x72, 0x76,
	0x65, 0x52, 0x65, 0x73, 0x75, 0x6c, 0x74, 0x42, 0x15, 0x5a, 0x13, 0x73, 0x69, 0x67, 0x6d, 0x61,
	0x6f, 0x73, 0x2f, 0x68, 0x6f, 0x74, 0x65, 0x6c, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x06,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_hotel_proto_reserve_proto_rawDescOnce sync.Once
	file_hotel_proto_reserve_proto_rawDescData = file_hotel_proto_reserve_proto_rawDesc
)

func file_hotel_proto_reserve_proto_rawDescGZIP() []byte {
	file_hotel_proto_reserve_proto_rawDescOnce.Do(func() {
		file_hotel_proto_reserve_proto_rawDescData = protoimpl.X.CompressGZIP(file_hotel_proto_reserve_proto_rawDescData)
	})
	return file_hotel_proto_reserve_proto_rawDescData
}

var file_hotel_proto_reserve_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_hotel_proto_reserve_proto_goTypes = []interface{}{
	(*ReserveRequest)(nil),          // 0: ReserveRequest
	(*ReserveResult)(nil),           // 1: ReserveResult
	(*proto.SpanContextConfig)(nil), // 2: SpanContextConfig
}
var file_hotel_proto_reserve_proto_depIdxs = []int32{
	2, // 0: ReserveRequest.spanContextConfig:type_name -> SpanContextConfig
	0, // 1: Reserve.MakeReservation:input_type -> ReserveRequest
	0, // 2: Reserve.CheckAvailability:input_type -> ReserveRequest
	1, // 3: Reserve.MakeReservation:output_type -> ReserveResult
	1, // 4: Reserve.CheckAvailability:output_type -> ReserveResult
	3, // [3:5] is the sub-list for method output_type
	1, // [1:3] is the sub-list for method input_type
	1, // [1:1] is the sub-list for extension type_name
	1, // [1:1] is the sub-list for extension extendee
	0, // [0:1] is the sub-list for field type_name
}

func init() { file_hotel_proto_reserve_proto_init() }
func file_hotel_proto_reserve_proto_init() {
	if File_hotel_proto_reserve_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_hotel_proto_reserve_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ReserveRequest); i {
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
		file_hotel_proto_reserve_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ReserveResult); i {
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
			RawDescriptor: file_hotel_proto_reserve_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_hotel_proto_reserve_proto_goTypes,
		DependencyIndexes: file_hotel_proto_reserve_proto_depIdxs,
		MessageInfos:      file_hotel_proto_reserve_proto_msgTypes,
	}.Build()
	File_hotel_proto_reserve_proto = out.File
	file_hotel_proto_reserve_proto_rawDesc = nil
	file_hotel_proto_reserve_proto_goTypes = nil
	file_hotel_proto_reserve_proto_depIdxs = nil
}
