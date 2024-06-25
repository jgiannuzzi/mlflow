// This proto file defines internal interfaces for MLflow, e.g
// enums used for storage in Tracking or Model Registry

// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.33.0
// 	protoc        v3.19.4
// source: internal.proto

package protos

import (
	_ "github.com/mlflow/mlflow/mlflow/go/pkg/protos/scalapb"
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

// Types of vertices represented in MLflow Run Inputs. Valid vertices are MLflow objects that can
// have an input relationship.
type InputVertexType int32

const (
	InputVertexType_RUN     InputVertexType = 1
	InputVertexType_DATASET InputVertexType = 2
)

// Enum value maps for InputVertexType.
var (
	InputVertexType_name = map[int32]string{
		1: "RUN",
		2: "DATASET",
	}
	InputVertexType_value = map[string]int32{
		"RUN":     1,
		"DATASET": 2,
	}
)

func (x InputVertexType) Enum() *InputVertexType {
	p := new(InputVertexType)
	*p = x
	return p
}

func (x InputVertexType) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (InputVertexType) Descriptor() protoreflect.EnumDescriptor {
	return file_internal_proto_enumTypes[0].Descriptor()
}

func (InputVertexType) Type() protoreflect.EnumType {
	return &file_internal_proto_enumTypes[0]
}

func (x InputVertexType) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Do not use.
func (x *InputVertexType) UnmarshalJSON(b []byte) error {
	num, err := protoimpl.X.UnmarshalJSONEnum(x.Descriptor(), b)
	if err != nil {
		return err
	}
	*x = InputVertexType(num)
	return nil
}

// Deprecated: Use InputVertexType.Descriptor instead.
func (InputVertexType) EnumDescriptor() ([]byte, []int) {
	return file_internal_proto_rawDescGZIP(), []int{0}
}

var File_internal_proto protoreflect.FileDescriptor

var file_internal_proto_rawDesc = []byte{
	0x0a, 0x0e, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x12, 0x0f, 0x6d, 0x6c, 0x66, 0x6c, 0x6f, 0x77, 0x2e, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61,
	0x6c, 0x1a, 0x15, 0x73, 0x63, 0x61, 0x6c, 0x61, 0x70, 0x62, 0x2f, 0x73, 0x63, 0x61, 0x6c, 0x61,
	0x70, 0x62, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2a, 0x27, 0x0a, 0x0f, 0x49, 0x6e, 0x70, 0x75,
	0x74, 0x56, 0x65, 0x72, 0x74, 0x65, 0x78, 0x54, 0x79, 0x70, 0x65, 0x12, 0x07, 0x0a, 0x03, 0x52,
	0x55, 0x4e, 0x10, 0x01, 0x12, 0x0b, 0x0a, 0x07, 0x44, 0x41, 0x54, 0x41, 0x53, 0x45, 0x54, 0x10,
	0x02, 0x42, 0x52, 0xe2, 0x3f, 0x02, 0x10, 0x01, 0x0a, 0x19, 0x6f, 0x72, 0x67, 0x2e, 0x6d, 0x6c,
	0x66, 0x6c, 0x6f, 0x77, 0x2e, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x5a, 0x2d, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f,
	0x6d, 0x6c, 0x66, 0x6c, 0x6f, 0x77, 0x2f, 0x6d, 0x6c, 0x66, 0x6c, 0x6f, 0x77, 0x2f, 0x6d, 0x6c,
	0x66, 0x6c, 0x6f, 0x77, 0x2f, 0x67, 0x6f, 0x2f, 0x70, 0x6b, 0x67, 0x2f, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x73, 0x90, 0x01, 0x01,
}

var (
	file_internal_proto_rawDescOnce sync.Once
	file_internal_proto_rawDescData = file_internal_proto_rawDesc
)

func file_internal_proto_rawDescGZIP() []byte {
	file_internal_proto_rawDescOnce.Do(func() {
		file_internal_proto_rawDescData = protoimpl.X.CompressGZIP(file_internal_proto_rawDescData)
	})
	return file_internal_proto_rawDescData
}

var file_internal_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_internal_proto_goTypes = []interface{}{
	(InputVertexType)(0), // 0: mlflow.internal.InputVertexType
}
var file_internal_proto_depIdxs = []int32{
	0, // [0:0] is the sub-list for method output_type
	0, // [0:0] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_internal_proto_init() }
func file_internal_proto_init() {
	if File_internal_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_internal_proto_rawDesc,
			NumEnums:      1,
			NumMessages:   0,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_internal_proto_goTypes,
		DependencyIndexes: file_internal_proto_depIdxs,
		EnumInfos:         file_internal_proto_enumTypes,
	}.Build()
	File_internal_proto = out.File
	file_internal_proto_rawDesc = nil
	file_internal_proto_goTypes = nil
	file_internal_proto_depIdxs = nil
}
