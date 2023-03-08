// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.28.1
// 	protoc        v3.21.12
// source: hotel/proto/prof.proto

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

type ProfRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	HotelIds []string `protobuf:"bytes,1,rep,name=hotelIds,proto3" json:"hotelIds,omitempty"`
	Locale   string   `protobuf:"bytes,2,opt,name=locale,proto3" json:"locale,omitempty"`
}

func (x *ProfRequest) Reset() {
	*x = ProfRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_hotel_proto_prof_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ProfRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ProfRequest) ProtoMessage() {}

func (x *ProfRequest) ProtoReflect() protoreflect.Message {
	mi := &file_hotel_proto_prof_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ProfRequest.ProtoReflect.Descriptor instead.
func (*ProfRequest) Descriptor() ([]byte, []int) {
	return file_hotel_proto_prof_proto_rawDescGZIP(), []int{0}
}

func (x *ProfRequest) GetHotelIds() []string {
	if x != nil {
		return x.HotelIds
	}
	return nil
}

func (x *ProfRequest) GetLocale() string {
	if x != nil {
		return x.Locale
	}
	return ""
}

type ProfileFlat struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	HotelId           string                   `protobuf:"bytes,1,opt,name=hotelId,proto3" json:"hotelId,omitempty"`
	Name              string                   `protobuf:"bytes,2,opt,name=name,proto3" json:"name,omitempty"`
	PhoneNumber       string                   `protobuf:"bytes,3,opt,name=phoneNumber,proto3" json:"phoneNumber,omitempty"`
	Description       string                   `protobuf:"bytes,4,opt,name=description,proto3" json:"description,omitempty"`
	StreetNumber      string                   `protobuf:"bytes,5,opt,name=streetNumber,proto3" json:"streetNumber,omitempty"`
	StreetName        string                   `protobuf:"bytes,6,opt,name=streetName,proto3" json:"streetName,omitempty"`
	City              string                   `protobuf:"bytes,7,opt,name=city,proto3" json:"city,omitempty"`
	State             string                   `protobuf:"bytes,8,opt,name=state,proto3" json:"state,omitempty"`
	Country           string                   `protobuf:"bytes,9,opt,name=country,proto3" json:"country,omitempty"`
	PostalCode        string                   `protobuf:"bytes,10,opt,name=postalCode,proto3" json:"postalCode,omitempty"`
	Lat               float32                  `protobuf:"fixed32,11,opt,name=lat,proto3" json:"lat,omitempty"`
	Lon               float32                  `protobuf:"fixed32,12,opt,name=lon,proto3" json:"lon,omitempty"`
	SpanContextConfig *proto.SpanContextConfig `protobuf:"bytes,13,opt,name=spanContextConfig,proto3" json:"spanContextConfig,omitempty"`
}

func (x *ProfileFlat) Reset() {
	*x = ProfileFlat{}
	if protoimpl.UnsafeEnabled {
		mi := &file_hotel_proto_prof_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ProfileFlat) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ProfileFlat) ProtoMessage() {}

func (x *ProfileFlat) ProtoReflect() protoreflect.Message {
	mi := &file_hotel_proto_prof_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ProfileFlat.ProtoReflect.Descriptor instead.
func (*ProfileFlat) Descriptor() ([]byte, []int) {
	return file_hotel_proto_prof_proto_rawDescGZIP(), []int{1}
}

func (x *ProfileFlat) GetHotelId() string {
	if x != nil {
		return x.HotelId
	}
	return ""
}

func (x *ProfileFlat) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *ProfileFlat) GetPhoneNumber() string {
	if x != nil {
		return x.PhoneNumber
	}
	return ""
}

func (x *ProfileFlat) GetDescription() string {
	if x != nil {
		return x.Description
	}
	return ""
}

func (x *ProfileFlat) GetStreetNumber() string {
	if x != nil {
		return x.StreetNumber
	}
	return ""
}

func (x *ProfileFlat) GetStreetName() string {
	if x != nil {
		return x.StreetName
	}
	return ""
}

func (x *ProfileFlat) GetCity() string {
	if x != nil {
		return x.City
	}
	return ""
}

func (x *ProfileFlat) GetState() string {
	if x != nil {
		return x.State
	}
	return ""
}

func (x *ProfileFlat) GetCountry() string {
	if x != nil {
		return x.Country
	}
	return ""
}

func (x *ProfileFlat) GetPostalCode() string {
	if x != nil {
		return x.PostalCode
	}
	return ""
}

func (x *ProfileFlat) GetLat() float32 {
	if x != nil {
		return x.Lat
	}
	return 0
}

func (x *ProfileFlat) GetLon() float32 {
	if x != nil {
		return x.Lon
	}
	return 0
}

func (x *ProfileFlat) GetSpanContextConfig() *proto.SpanContextConfig {
	if x != nil {
		return x.SpanContextConfig
	}
	return nil
}

type ProfResult struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Hotels []*ProfileFlat `protobuf:"bytes,1,rep,name=hotels,proto3" json:"hotels,omitempty"`
}

func (x *ProfResult) Reset() {
	*x = ProfResult{}
	if protoimpl.UnsafeEnabled {
		mi := &file_hotel_proto_prof_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ProfResult) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ProfResult) ProtoMessage() {}

func (x *ProfResult) ProtoReflect() protoreflect.Message {
	mi := &file_hotel_proto_prof_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ProfResult.ProtoReflect.Descriptor instead.
func (*ProfResult) Descriptor() ([]byte, []int) {
	return file_hotel_proto_prof_proto_rawDescGZIP(), []int{2}
}

func (x *ProfResult) GetHotels() []*ProfileFlat {
	if x != nil {
		return x.Hotels
	}
	return nil
}

var File_hotel_proto_prof_proto protoreflect.FileDescriptor

var file_hotel_proto_prof_proto_rawDesc = []byte{
	0x0a, 0x16, 0x68, 0x6f, 0x74, 0x65, 0x6c, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x70, 0x72,
	0x6f, 0x66, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x1b, 0x74, 0x72, 0x61, 0x63, 0x69, 0x6e,
	0x67, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x74, 0x72, 0x61, 0x63, 0x69, 0x6e, 0x67, 0x2e,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x41, 0x0a, 0x0b, 0x50, 0x72, 0x6f, 0x66, 0x52, 0x65, 0x71,
	0x75, 0x65, 0x73, 0x74, 0x12, 0x1a, 0x0a, 0x08, 0x68, 0x6f, 0x74, 0x65, 0x6c, 0x49, 0x64, 0x73,
	0x18, 0x01, 0x20, 0x03, 0x28, 0x09, 0x52, 0x08, 0x68, 0x6f, 0x74, 0x65, 0x6c, 0x49, 0x64, 0x73,
	0x12, 0x16, 0x0a, 0x06, 0x6c, 0x6f, 0x63, 0x61, 0x6c, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x06, 0x6c, 0x6f, 0x63, 0x61, 0x6c, 0x65, 0x22, 0x8d, 0x03, 0x0a, 0x0b, 0x50, 0x72, 0x6f,
	0x66, 0x69, 0x6c, 0x65, 0x46, 0x6c, 0x61, 0x74, 0x12, 0x18, 0x0a, 0x07, 0x68, 0x6f, 0x74, 0x65,
	0x6c, 0x49, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x68, 0x6f, 0x74, 0x65, 0x6c,
	0x49, 0x64, 0x12, 0x12, 0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x12, 0x20, 0x0a, 0x0b, 0x70, 0x68, 0x6f, 0x6e, 0x65, 0x4e,
	0x75, 0x6d, 0x62, 0x65, 0x72, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0b, 0x70, 0x68, 0x6f,
	0x6e, 0x65, 0x4e, 0x75, 0x6d, 0x62, 0x65, 0x72, 0x12, 0x20, 0x0a, 0x0b, 0x64, 0x65, 0x73, 0x63,
	0x72, 0x69, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x18, 0x04, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0b, 0x64,
	0x65, 0x73, 0x63, 0x72, 0x69, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x12, 0x22, 0x0a, 0x0c, 0x73, 0x74,
	0x72, 0x65, 0x65, 0x74, 0x4e, 0x75, 0x6d, 0x62, 0x65, 0x72, 0x18, 0x05, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x0c, 0x73, 0x74, 0x72, 0x65, 0x65, 0x74, 0x4e, 0x75, 0x6d, 0x62, 0x65, 0x72, 0x12, 0x1e,
	0x0a, 0x0a, 0x73, 0x74, 0x72, 0x65, 0x65, 0x74, 0x4e, 0x61, 0x6d, 0x65, 0x18, 0x06, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x0a, 0x73, 0x74, 0x72, 0x65, 0x65, 0x74, 0x4e, 0x61, 0x6d, 0x65, 0x12, 0x12,
	0x0a, 0x04, 0x63, 0x69, 0x74, 0x79, 0x18, 0x07, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x63, 0x69,
	0x74, 0x79, 0x12, 0x14, 0x0a, 0x05, 0x73, 0x74, 0x61, 0x74, 0x65, 0x18, 0x08, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x05, 0x73, 0x74, 0x61, 0x74, 0x65, 0x12, 0x18, 0x0a, 0x07, 0x63, 0x6f, 0x75, 0x6e,
	0x74, 0x72, 0x79, 0x18, 0x09, 0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x63, 0x6f, 0x75, 0x6e, 0x74,
	0x72, 0x79, 0x12, 0x1e, 0x0a, 0x0a, 0x70, 0x6f, 0x73, 0x74, 0x61, 0x6c, 0x43, 0x6f, 0x64, 0x65,
	0x18, 0x0a, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0a, 0x70, 0x6f, 0x73, 0x74, 0x61, 0x6c, 0x43, 0x6f,
	0x64, 0x65, 0x12, 0x10, 0x0a, 0x03, 0x6c, 0x61, 0x74, 0x18, 0x0b, 0x20, 0x01, 0x28, 0x02, 0x52,
	0x03, 0x6c, 0x61, 0x74, 0x12, 0x10, 0x0a, 0x03, 0x6c, 0x6f, 0x6e, 0x18, 0x0c, 0x20, 0x01, 0x28,
	0x02, 0x52, 0x03, 0x6c, 0x6f, 0x6e, 0x12, 0x40, 0x0a, 0x11, 0x73, 0x70, 0x61, 0x6e, 0x43, 0x6f,
	0x6e, 0x74, 0x65, 0x78, 0x74, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x18, 0x0d, 0x20, 0x01, 0x28,
	0x0b, 0x32, 0x12, 0x2e, 0x53, 0x70, 0x61, 0x6e, 0x43, 0x6f, 0x6e, 0x74, 0x65, 0x78, 0x74, 0x43,
	0x6f, 0x6e, 0x66, 0x69, 0x67, 0x52, 0x11, 0x73, 0x70, 0x61, 0x6e, 0x43, 0x6f, 0x6e, 0x74, 0x65,
	0x78, 0x74, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x22, 0x32, 0x0a, 0x0a, 0x50, 0x72, 0x6f, 0x66,
	0x52, 0x65, 0x73, 0x75, 0x6c, 0x74, 0x12, 0x24, 0x0a, 0x06, 0x68, 0x6f, 0x74, 0x65, 0x6c, 0x73,
	0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x0c, 0x2e, 0x50, 0x72, 0x6f, 0x66, 0x69, 0x6c, 0x65,
	0x46, 0x6c, 0x61, 0x74, 0x52, 0x06, 0x68, 0x6f, 0x74, 0x65, 0x6c, 0x73, 0x32, 0x30, 0x0a, 0x04,
	0x50, 0x72, 0x6f, 0x66, 0x12, 0x28, 0x0a, 0x0b, 0x47, 0x65, 0x74, 0x50, 0x72, 0x6f, 0x66, 0x69,
	0x6c, 0x65, 0x73, 0x12, 0x0c, 0x2e, 0x50, 0x72, 0x6f, 0x66, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73,
	0x74, 0x1a, 0x0b, 0x2e, 0x50, 0x72, 0x6f, 0x66, 0x52, 0x65, 0x73, 0x75, 0x6c, 0x74, 0x42, 0x15,
	0x5a, 0x13, 0x73, 0x69, 0x67, 0x6d, 0x61, 0x6f, 0x73, 0x2f, 0x68, 0x6f, 0x74, 0x65, 0x6c, 0x2f,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_hotel_proto_prof_proto_rawDescOnce sync.Once
	file_hotel_proto_prof_proto_rawDescData = file_hotel_proto_prof_proto_rawDesc
)

func file_hotel_proto_prof_proto_rawDescGZIP() []byte {
	file_hotel_proto_prof_proto_rawDescOnce.Do(func() {
		file_hotel_proto_prof_proto_rawDescData = protoimpl.X.CompressGZIP(file_hotel_proto_prof_proto_rawDescData)
	})
	return file_hotel_proto_prof_proto_rawDescData
}

var file_hotel_proto_prof_proto_msgTypes = make([]protoimpl.MessageInfo, 3)
var file_hotel_proto_prof_proto_goTypes = []interface{}{
	(*ProfRequest)(nil),             // 0: ProfRequest
	(*ProfileFlat)(nil),             // 1: ProfileFlat
	(*ProfResult)(nil),              // 2: ProfResult
	(*proto.SpanContextConfig)(nil), // 3: SpanContextConfig
}
var file_hotel_proto_prof_proto_depIdxs = []int32{
	3, // 0: ProfileFlat.spanContextConfig:type_name -> SpanContextConfig
	1, // 1: ProfResult.hotels:type_name -> ProfileFlat
	0, // 2: Prof.GetProfiles:input_type -> ProfRequest
	2, // 3: Prof.GetProfiles:output_type -> ProfResult
	3, // [3:4] is the sub-list for method output_type
	2, // [2:3] is the sub-list for method input_type
	2, // [2:2] is the sub-list for extension type_name
	2, // [2:2] is the sub-list for extension extendee
	0, // [0:2] is the sub-list for field type_name
}

func init() { file_hotel_proto_prof_proto_init() }
func file_hotel_proto_prof_proto_init() {
	if File_hotel_proto_prof_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_hotel_proto_prof_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ProfRequest); i {
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
		file_hotel_proto_prof_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ProfileFlat); i {
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
		file_hotel_proto_prof_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ProfResult); i {
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
			RawDescriptor: file_hotel_proto_prof_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   3,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_hotel_proto_prof_proto_goTypes,
		DependencyIndexes: file_hotel_proto_prof_proto_depIdxs,
		MessageInfos:      file_hotel_proto_prof_proto_msgTypes,
	}.Build()
	File_hotel_proto_prof_proto = out.File
	file_hotel_proto_prof_proto_rawDesc = nil
	file_hotel_proto_prof_proto_goTypes = nil
	file_hotel_proto_prof_proto_depIdxs = nil
}
