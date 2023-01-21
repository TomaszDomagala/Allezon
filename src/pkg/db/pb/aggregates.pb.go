// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.28.1
// 	protoc        v3.21.12
// source: aggregates.proto

package pb

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

type ActionAggregate struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	SerializedKey uint32 `protobuf:"varint,1,opt,name=SerializedKey,proto3" json:"SerializedKey,omitempty"`
	Data          uint32 `protobuf:"varint,2,opt,name=Data,proto3" json:"Data,omitempty"`
}

func (x *ActionAggregate) Reset() {
	*x = ActionAggregate{}
	if protoimpl.UnsafeEnabled {
		mi := &file_aggregates_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ActionAggregate) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ActionAggregate) ProtoMessage() {}

func (x *ActionAggregate) ProtoReflect() protoreflect.Message {
	mi := &file_aggregates_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ActionAggregate.ProtoReflect.Descriptor instead.
func (*ActionAggregate) Descriptor() ([]byte, []int) {
	return file_aggregates_proto_rawDescGZIP(), []int{0}
}

func (x *ActionAggregate) GetSerializedKey() uint32 {
	if x != nil {
		return x.SerializedKey
	}
	return 0
}

func (x *ActionAggregate) GetData() uint32 {
	if x != nil {
		return x.Data
	}
	return 0
}

type TypeAggregate struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Count []*ActionAggregate `protobuf:"bytes,1,rep,name=Count,proto3" json:"Count,omitempty"`
	Sum   []*ActionAggregate `protobuf:"bytes,2,rep,name=Sum,proto3" json:"Sum,omitempty"`
}

func (x *TypeAggregate) Reset() {
	*x = TypeAggregate{}
	if protoimpl.UnsafeEnabled {
		mi := &file_aggregates_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *TypeAggregate) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*TypeAggregate) ProtoMessage() {}

func (x *TypeAggregate) ProtoReflect() protoreflect.Message {
	mi := &file_aggregates_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use TypeAggregate.ProtoReflect.Descriptor instead.
func (*TypeAggregate) Descriptor() ([]byte, []int) {
	return file_aggregates_proto_rawDescGZIP(), []int{1}
}

func (x *TypeAggregate) GetCount() []*ActionAggregate {
	if x != nil {
		return x.Count
	}
	return nil
}

func (x *TypeAggregate) GetSum() []*ActionAggregate {
	if x != nil {
		return x.Sum
	}
	return nil
}

var File_aggregates_proto protoreflect.FileDescriptor

var file_aggregates_proto_rawDesc = []byte{
	0x0a, 0x10, 0x61, 0x67, 0x67, 0x72, 0x65, 0x67, 0x61, 0x74, 0x65, 0x73, 0x2e, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x12, 0x0a, 0x61, 0x67, 0x67, 0x72, 0x65, 0x67, 0x61, 0x74, 0x65, 0x73, 0x22, 0x4b,
	0x0a, 0x0f, 0x41, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x41, 0x67, 0x67, 0x72, 0x65, 0x67, 0x61, 0x74,
	0x65, 0x12, 0x24, 0x0a, 0x0d, 0x53, 0x65, 0x72, 0x69, 0x61, 0x6c, 0x69, 0x7a, 0x65, 0x64, 0x4b,
	0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x0d, 0x53, 0x65, 0x72, 0x69, 0x61, 0x6c,
	0x69, 0x7a, 0x65, 0x64, 0x4b, 0x65, 0x79, 0x12, 0x12, 0x0a, 0x04, 0x44, 0x61, 0x74, 0x61, 0x18,
	0x02, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x04, 0x44, 0x61, 0x74, 0x61, 0x22, 0x71, 0x0a, 0x0d, 0x54,
	0x79, 0x70, 0x65, 0x41, 0x67, 0x67, 0x72, 0x65, 0x67, 0x61, 0x74, 0x65, 0x12, 0x31, 0x0a, 0x05,
	0x43, 0x6f, 0x75, 0x6e, 0x74, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x1b, 0x2e, 0x61, 0x67,
	0x67, 0x72, 0x65, 0x67, 0x61, 0x74, 0x65, 0x73, 0x2e, 0x41, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x41,
	0x67, 0x67, 0x72, 0x65, 0x67, 0x61, 0x74, 0x65, 0x52, 0x05, 0x43, 0x6f, 0x75, 0x6e, 0x74, 0x12,
	0x2d, 0x0a, 0x03, 0x53, 0x75, 0x6d, 0x18, 0x02, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x1b, 0x2e, 0x61,
	0x67, 0x67, 0x72, 0x65, 0x67, 0x61, 0x74, 0x65, 0x73, 0x2e, 0x41, 0x63, 0x74, 0x69, 0x6f, 0x6e,
	0x41, 0x67, 0x67, 0x72, 0x65, 0x67, 0x61, 0x74, 0x65, 0x52, 0x03, 0x53, 0x75, 0x6d, 0x42, 0x06,
	0x5a, 0x04, 0x2e, 0x2f, 0x70, 0x62, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_aggregates_proto_rawDescOnce sync.Once
	file_aggregates_proto_rawDescData = file_aggregates_proto_rawDesc
)

func file_aggregates_proto_rawDescGZIP() []byte {
	file_aggregates_proto_rawDescOnce.Do(func() {
		file_aggregates_proto_rawDescData = protoimpl.X.CompressGZIP(file_aggregates_proto_rawDescData)
	})
	return file_aggregates_proto_rawDescData
}

var file_aggregates_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_aggregates_proto_goTypes = []interface{}{
	(*ActionAggregate)(nil), // 0: aggregates.ActionAggregate
	(*TypeAggregate)(nil),   // 1: aggregates.TypeAggregate
}
var file_aggregates_proto_depIdxs = []int32{
	0, // 0: aggregates.TypeAggregate.Count:type_name -> aggregates.ActionAggregate
	0, // 1: aggregates.TypeAggregate.Sum:type_name -> aggregates.ActionAggregate
	2, // [2:2] is the sub-list for method output_type
	2, // [2:2] is the sub-list for method input_type
	2, // [2:2] is the sub-list for extension type_name
	2, // [2:2] is the sub-list for extension extendee
	0, // [0:2] is the sub-list for field type_name
}

func init() { file_aggregates_proto_init() }
func file_aggregates_proto_init() {
	if File_aggregates_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_aggregates_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ActionAggregate); i {
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
		file_aggregates_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*TypeAggregate); i {
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
			RawDescriptor: file_aggregates_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_aggregates_proto_goTypes,
		DependencyIndexes: file_aggregates_proto_depIdxs,
		MessageInfos:      file_aggregates_proto_msgTypes,
	}.Build()
	File_aggregates_proto = out.File
	file_aggregates_proto_rawDesc = nil
	file_aggregates_proto_goTypes = nil
	file_aggregates_proto_depIdxs = nil
}
