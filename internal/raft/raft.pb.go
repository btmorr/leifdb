// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.23.0
// 	protoc        v3.12.1
// source: raft.proto

package raft

import (
	context "context"
	proto "github.com/golang/protobuf/proto"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
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

// This is a compile-time assertion that a sufficiently up-to-date version
// of the legacy proto package is being used.
const _ = proto.ProtoPackageIsVersion4

type LogRecord_Action int32

const (
	LogRecord_SET LogRecord_Action = 0
	LogRecord_DEL LogRecord_Action = 1
)

// Enum value maps for LogRecord_Action.
var (
	LogRecord_Action_name = map[int32]string{
		0: "SET",
		1: "DEL",
	}
	LogRecord_Action_value = map[string]int32{
		"SET": 0,
		"DEL": 1,
	}
)

func (x LogRecord_Action) Enum() *LogRecord_Action {
	p := new(LogRecord_Action)
	*p = x
	return p
}

func (x LogRecord_Action) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (LogRecord_Action) Descriptor() protoreflect.EnumDescriptor {
	return file_raft_proto_enumTypes[0].Descriptor()
}

func (LogRecord_Action) Type() protoreflect.EnumType {
	return &file_raft_proto_enumTypes[0]
}

func (x LogRecord_Action) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use LogRecord_Action.Descriptor instead.
func (LogRecord_Action) EnumDescriptor() ([]byte, []int) {
	return file_raft_proto_rawDescGZIP(), []int{4, 0}
}

type VoteRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Term         int64  `protobuf:"varint,1,opt,name=term,proto3" json:"term,omitempty"`
	CandidateID  string `protobuf:"bytes,2,opt,name=candidateID,proto3" json:"candidateID,omitempty"`
	LastLogIndex int64  `protobuf:"varint,3,opt,name=lastLogIndex,proto3" json:"lastLogIndex,omitempty"`
	LastLogTerm  int64  `protobuf:"varint,4,opt,name=lastLogTerm,proto3" json:"lastLogTerm,omitempty"`
}

func (x *VoteRequest) Reset() {
	*x = VoteRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_raft_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *VoteRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*VoteRequest) ProtoMessage() {}

func (x *VoteRequest) ProtoReflect() protoreflect.Message {
	mi := &file_raft_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use VoteRequest.ProtoReflect.Descriptor instead.
func (*VoteRequest) Descriptor() ([]byte, []int) {
	return file_raft_proto_rawDescGZIP(), []int{0}
}

func (x *VoteRequest) GetTerm() int64 {
	if x != nil {
		return x.Term
	}
	return 0
}

func (x *VoteRequest) GetCandidateID() string {
	if x != nil {
		return x.CandidateID
	}
	return ""
}

func (x *VoteRequest) GetLastLogIndex() int64 {
	if x != nil {
		return x.LastLogIndex
	}
	return 0
}

func (x *VoteRequest) GetLastLogTerm() int64 {
	if x != nil {
		return x.LastLogTerm
	}
	return 0
}

type VoteReply struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Term        int64 `protobuf:"varint,1,opt,name=term,proto3" json:"term,omitempty"`
	VoteGranted bool  `protobuf:"varint,2,opt,name=voteGranted,proto3" json:"voteGranted,omitempty"`
}

func (x *VoteReply) Reset() {
	*x = VoteReply{}
	if protoimpl.UnsafeEnabled {
		mi := &file_raft_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *VoteReply) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*VoteReply) ProtoMessage() {}

func (x *VoteReply) ProtoReflect() protoreflect.Message {
	mi := &file_raft_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use VoteReply.ProtoReflect.Descriptor instead.
func (*VoteReply) Descriptor() ([]byte, []int) {
	return file_raft_proto_rawDescGZIP(), []int{1}
}

func (x *VoteReply) GetTerm() int64 {
	if x != nil {
		return x.Term
	}
	return 0
}

func (x *VoteReply) GetVoteGranted() bool {
	if x != nil {
		return x.VoteGranted
	}
	return false
}

type AppendRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Term         int64        `protobuf:"varint,1,opt,name=term,proto3" json:"term,omitempty"`
	LeaderID     string       `protobuf:"bytes,2,opt,name=leaderID,proto3" json:"leaderID,omitempty"`
	PrevLogIndex int64        `protobuf:"varint,3,opt,name=prevLogIndex,proto3" json:"prevLogIndex,omitempty"`
	LeaderCommit int64        `protobuf:"varint,4,opt,name=leaderCommit,proto3" json:"leaderCommit,omitempty"`
	Entries      []*LogRecord `protobuf:"bytes,5,rep,name=entries,proto3" json:"entries,omitempty"`
}

func (x *AppendRequest) Reset() {
	*x = AppendRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_raft_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *AppendRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*AppendRequest) ProtoMessage() {}

func (x *AppendRequest) ProtoReflect() protoreflect.Message {
	mi := &file_raft_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use AppendRequest.ProtoReflect.Descriptor instead.
func (*AppendRequest) Descriptor() ([]byte, []int) {
	return file_raft_proto_rawDescGZIP(), []int{2}
}

func (x *AppendRequest) GetTerm() int64 {
	if x != nil {
		return x.Term
	}
	return 0
}

func (x *AppendRequest) GetLeaderID() string {
	if x != nil {
		return x.LeaderID
	}
	return ""
}

func (x *AppendRequest) GetPrevLogIndex() int64 {
	if x != nil {
		return x.PrevLogIndex
	}
	return 0
}

func (x *AppendRequest) GetLeaderCommit() int64 {
	if x != nil {
		return x.LeaderCommit
	}
	return 0
}

func (x *AppendRequest) GetEntries() []*LogRecord {
	if x != nil {
		return x.Entries
	}
	return nil
}

type AppendReply struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Term    int64 `protobuf:"varint,1,opt,name=term,proto3" json:"term,omitempty"`
	Success bool  `protobuf:"varint,2,opt,name=success,proto3" json:"success,omitempty"`
}

func (x *AppendReply) Reset() {
	*x = AppendReply{}
	if protoimpl.UnsafeEnabled {
		mi := &file_raft_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *AppendReply) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*AppendReply) ProtoMessage() {}

func (x *AppendReply) ProtoReflect() protoreflect.Message {
	mi := &file_raft_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use AppendReply.ProtoReflect.Descriptor instead.
func (*AppendReply) Descriptor() ([]byte, []int) {
	return file_raft_proto_rawDescGZIP(), []int{3}
}

func (x *AppendReply) GetTerm() int64 {
	if x != nil {
		return x.Term
	}
	return 0
}

func (x *AppendReply) GetSuccess() bool {
	if x != nil {
		return x.Success
	}
	return false
}

type LogRecord struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Term   int64            `protobuf:"varint,1,opt,name=term,proto3" json:"term,omitempty"`
	Action LogRecord_Action `protobuf:"varint,2,opt,name=action,proto3,enum=raft.LogRecord_Action" json:"action,omitempty"`
	Key    string           `protobuf:"bytes,3,opt,name=key,proto3" json:"key,omitempty"`
	Value  string           `protobuf:"bytes,4,opt,name=value,proto3" json:"value,omitempty"`
}

func (x *LogRecord) Reset() {
	*x = LogRecord{}
	if protoimpl.UnsafeEnabled {
		mi := &file_raft_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *LogRecord) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*LogRecord) ProtoMessage() {}

func (x *LogRecord) ProtoReflect() protoreflect.Message {
	mi := &file_raft_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use LogRecord.ProtoReflect.Descriptor instead.
func (*LogRecord) Descriptor() ([]byte, []int) {
	return file_raft_proto_rawDescGZIP(), []int{4}
}

func (x *LogRecord) GetTerm() int64 {
	if x != nil {
		return x.Term
	}
	return 0
}

func (x *LogRecord) GetAction() LogRecord_Action {
	if x != nil {
		return x.Action
	}
	return LogRecord_SET
}

func (x *LogRecord) GetKey() string {
	if x != nil {
		return x.Key
	}
	return ""
}

func (x *LogRecord) GetValue() string {
	if x != nil {
		return x.Value
	}
	return ""
}

var File_raft_proto protoreflect.FileDescriptor

var file_raft_proto_rawDesc = []byte{
	0x0a, 0x0a, 0x72, 0x61, 0x66, 0x74, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x04, 0x72, 0x61,
	0x66, 0x74, 0x22, 0x89, 0x01, 0x0a, 0x0b, 0x56, 0x6f, 0x74, 0x65, 0x52, 0x65, 0x71, 0x75, 0x65,
	0x73, 0x74, 0x12, 0x12, 0x0a, 0x04, 0x74, 0x65, 0x72, 0x6d, 0x18, 0x01, 0x20, 0x01, 0x28, 0x03,
	0x52, 0x04, 0x74, 0x65, 0x72, 0x6d, 0x12, 0x20, 0x0a, 0x0b, 0x63, 0x61, 0x6e, 0x64, 0x69, 0x64,
	0x61, 0x74, 0x65, 0x49, 0x44, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0b, 0x63, 0x61, 0x6e,
	0x64, 0x69, 0x64, 0x61, 0x74, 0x65, 0x49, 0x44, 0x12, 0x22, 0x0a, 0x0c, 0x6c, 0x61, 0x73, 0x74,
	0x4c, 0x6f, 0x67, 0x49, 0x6e, 0x64, 0x65, 0x78, 0x18, 0x03, 0x20, 0x01, 0x28, 0x03, 0x52, 0x0c,
	0x6c, 0x61, 0x73, 0x74, 0x4c, 0x6f, 0x67, 0x49, 0x6e, 0x64, 0x65, 0x78, 0x12, 0x20, 0x0a, 0x0b,
	0x6c, 0x61, 0x73, 0x74, 0x4c, 0x6f, 0x67, 0x54, 0x65, 0x72, 0x6d, 0x18, 0x04, 0x20, 0x01, 0x28,
	0x03, 0x52, 0x0b, 0x6c, 0x61, 0x73, 0x74, 0x4c, 0x6f, 0x67, 0x54, 0x65, 0x72, 0x6d, 0x22, 0x41,
	0x0a, 0x09, 0x56, 0x6f, 0x74, 0x65, 0x52, 0x65, 0x70, 0x6c, 0x79, 0x12, 0x12, 0x0a, 0x04, 0x74,
	0x65, 0x72, 0x6d, 0x18, 0x01, 0x20, 0x01, 0x28, 0x03, 0x52, 0x04, 0x74, 0x65, 0x72, 0x6d, 0x12,
	0x20, 0x0a, 0x0b, 0x76, 0x6f, 0x74, 0x65, 0x47, 0x72, 0x61, 0x6e, 0x74, 0x65, 0x64, 0x18, 0x02,
	0x20, 0x01, 0x28, 0x08, 0x52, 0x0b, 0x76, 0x6f, 0x74, 0x65, 0x47, 0x72, 0x61, 0x6e, 0x74, 0x65,
	0x64, 0x22, 0xb2, 0x01, 0x0a, 0x0d, 0x41, 0x70, 0x70, 0x65, 0x6e, 0x64, 0x52, 0x65, 0x71, 0x75,
	0x65, 0x73, 0x74, 0x12, 0x12, 0x0a, 0x04, 0x74, 0x65, 0x72, 0x6d, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x03, 0x52, 0x04, 0x74, 0x65, 0x72, 0x6d, 0x12, 0x1a, 0x0a, 0x08, 0x6c, 0x65, 0x61, 0x64, 0x65,
	0x72, 0x49, 0x44, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x08, 0x6c, 0x65, 0x61, 0x64, 0x65,
	0x72, 0x49, 0x44, 0x12, 0x22, 0x0a, 0x0c, 0x70, 0x72, 0x65, 0x76, 0x4c, 0x6f, 0x67, 0x49, 0x6e,
	0x64, 0x65, 0x78, 0x18, 0x03, 0x20, 0x01, 0x28, 0x03, 0x52, 0x0c, 0x70, 0x72, 0x65, 0x76, 0x4c,
	0x6f, 0x67, 0x49, 0x6e, 0x64, 0x65, 0x78, 0x12, 0x22, 0x0a, 0x0c, 0x6c, 0x65, 0x61, 0x64, 0x65,
	0x72, 0x43, 0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x18, 0x04, 0x20, 0x01, 0x28, 0x03, 0x52, 0x0c, 0x6c,
	0x65, 0x61, 0x64, 0x65, 0x72, 0x43, 0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x12, 0x29, 0x0a, 0x07, 0x65,
	0x6e, 0x74, 0x72, 0x69, 0x65, 0x73, 0x18, 0x05, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x0f, 0x2e, 0x72,
	0x61, 0x66, 0x74, 0x2e, 0x4c, 0x6f, 0x67, 0x52, 0x65, 0x63, 0x6f, 0x72, 0x64, 0x52, 0x07, 0x65,
	0x6e, 0x74, 0x72, 0x69, 0x65, 0x73, 0x22, 0x3b, 0x0a, 0x0b, 0x41, 0x70, 0x70, 0x65, 0x6e, 0x64,
	0x52, 0x65, 0x70, 0x6c, 0x79, 0x12, 0x12, 0x0a, 0x04, 0x74, 0x65, 0x72, 0x6d, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x03, 0x52, 0x04, 0x74, 0x65, 0x72, 0x6d, 0x12, 0x18, 0x0a, 0x07, 0x73, 0x75, 0x63,
	0x63, 0x65, 0x73, 0x73, 0x18, 0x02, 0x20, 0x01, 0x28, 0x08, 0x52, 0x07, 0x73, 0x75, 0x63, 0x63,
	0x65, 0x73, 0x73, 0x22, 0x93, 0x01, 0x0a, 0x09, 0x4c, 0x6f, 0x67, 0x52, 0x65, 0x63, 0x6f, 0x72,
	0x64, 0x12, 0x12, 0x0a, 0x04, 0x74, 0x65, 0x72, 0x6d, 0x18, 0x01, 0x20, 0x01, 0x28, 0x03, 0x52,
	0x04, 0x74, 0x65, 0x72, 0x6d, 0x12, 0x2e, 0x0a, 0x06, 0x61, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x18,
	0x02, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x16, 0x2e, 0x72, 0x61, 0x66, 0x74, 0x2e, 0x4c, 0x6f, 0x67,
	0x52, 0x65, 0x63, 0x6f, 0x72, 0x64, 0x2e, 0x41, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x06, 0x61,
	0x63, 0x74, 0x69, 0x6f, 0x6e, 0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x03, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x03, 0x6b, 0x65, 0x79, 0x12, 0x14, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65,
	0x18, 0x04, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x22, 0x1a, 0x0a,
	0x06, 0x41, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x12, 0x07, 0x0a, 0x03, 0x53, 0x45, 0x54, 0x10, 0x00,
	0x12, 0x07, 0x0a, 0x03, 0x44, 0x45, 0x4c, 0x10, 0x01, 0x32, 0x73, 0x0a, 0x04, 0x52, 0x61, 0x66,
	0x74, 0x12, 0x33, 0x0a, 0x0b, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x56, 0x6f, 0x74, 0x65,
	0x12, 0x11, 0x2e, 0x72, 0x61, 0x66, 0x74, 0x2e, 0x56, 0x6f, 0x74, 0x65, 0x52, 0x65, 0x71, 0x75,
	0x65, 0x73, 0x74, 0x1a, 0x0f, 0x2e, 0x72, 0x61, 0x66, 0x74, 0x2e, 0x56, 0x6f, 0x74, 0x65, 0x52,
	0x65, 0x70, 0x6c, 0x79, 0x22, 0x00, 0x12, 0x36, 0x0a, 0x0a, 0x41, 0x70, 0x70, 0x65, 0x6e, 0x64,
	0x4c, 0x6f, 0x67, 0x73, 0x12, 0x13, 0x2e, 0x72, 0x61, 0x66, 0x74, 0x2e, 0x41, 0x70, 0x70, 0x65,
	0x6e, 0x64, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x11, 0x2e, 0x72, 0x61, 0x66, 0x74,
	0x2e, 0x41, 0x70, 0x70, 0x65, 0x6e, 0x64, 0x52, 0x65, 0x70, 0x6c, 0x79, 0x22, 0x00, 0x42, 0x28,
	0x5a, 0x26, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x62, 0x74, 0x6d,
	0x6f, 0x72, 0x72, 0x2f, 0x6c, 0x65, 0x69, 0x66, 0x64, 0x62, 0x2f, 0x69, 0x6e, 0x74, 0x65, 0x72,
	0x6e, 0x61, 0x6c, 0x2f, 0x72, 0x61, 0x66, 0x74, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_raft_proto_rawDescOnce sync.Once
	file_raft_proto_rawDescData = file_raft_proto_rawDesc
)

func file_raft_proto_rawDescGZIP() []byte {
	file_raft_proto_rawDescOnce.Do(func() {
		file_raft_proto_rawDescData = protoimpl.X.CompressGZIP(file_raft_proto_rawDescData)
	})
	return file_raft_proto_rawDescData
}

var file_raft_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_raft_proto_msgTypes = make([]protoimpl.MessageInfo, 5)
var file_raft_proto_goTypes = []interface{}{
	(LogRecord_Action)(0), // 0: raft.LogRecord.Action
	(*VoteRequest)(nil),   // 1: raft.VoteRequest
	(*VoteReply)(nil),     // 2: raft.VoteReply
	(*AppendRequest)(nil), // 3: raft.AppendRequest
	(*AppendReply)(nil),   // 4: raft.AppendReply
	(*LogRecord)(nil),     // 5: raft.LogRecord
}
var file_raft_proto_depIdxs = []int32{
	5, // 0: raft.AppendRequest.entries:type_name -> raft.LogRecord
	0, // 1: raft.LogRecord.action:type_name -> raft.LogRecord.Action
	1, // 2: raft.Raft.RequestVote:input_type -> raft.VoteRequest
	3, // 3: raft.Raft.AppendLogs:input_type -> raft.AppendRequest
	2, // 4: raft.Raft.RequestVote:output_type -> raft.VoteReply
	4, // 5: raft.Raft.AppendLogs:output_type -> raft.AppendReply
	4, // [4:6] is the sub-list for method output_type
	2, // [2:4] is the sub-list for method input_type
	2, // [2:2] is the sub-list for extension type_name
	2, // [2:2] is the sub-list for extension extendee
	0, // [0:2] is the sub-list for field type_name
}

func init() { file_raft_proto_init() }
func file_raft_proto_init() {
	if File_raft_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_raft_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*VoteRequest); i {
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
		file_raft_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*VoteReply); i {
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
		file_raft_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*AppendRequest); i {
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
		file_raft_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*AppendReply); i {
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
		file_raft_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*LogRecord); i {
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
			RawDescriptor: file_raft_proto_rawDesc,
			NumEnums:      1,
			NumMessages:   5,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_raft_proto_goTypes,
		DependencyIndexes: file_raft_proto_depIdxs,
		EnumInfos:         file_raft_proto_enumTypes,
		MessageInfos:      file_raft_proto_msgTypes,
	}.Build()
	File_raft_proto = out.File
	file_raft_proto_rawDesc = nil
	file_raft_proto_goTypes = nil
	file_raft_proto_depIdxs = nil
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConnInterface

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion6

// RaftClient is the client API for Raft service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type RaftClient interface {
	RequestVote(ctx context.Context, in *VoteRequest, opts ...grpc.CallOption) (*VoteReply, error)
	AppendLogs(ctx context.Context, in *AppendRequest, opts ...grpc.CallOption) (*AppendReply, error)
}

type raftClient struct {
	cc grpc.ClientConnInterface
}

func NewRaftClient(cc grpc.ClientConnInterface) RaftClient {
	return &raftClient{cc}
}

func (c *raftClient) RequestVote(ctx context.Context, in *VoteRequest, opts ...grpc.CallOption) (*VoteReply, error) {
	out := new(VoteReply)
	err := c.cc.Invoke(ctx, "/raft.Raft/RequestVote", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *raftClient) AppendLogs(ctx context.Context, in *AppendRequest, opts ...grpc.CallOption) (*AppendReply, error) {
	out := new(AppendReply)
	err := c.cc.Invoke(ctx, "/raft.Raft/AppendLogs", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// RaftServer is the server API for Raft service.
type RaftServer interface {
	RequestVote(context.Context, *VoteRequest) (*VoteReply, error)
	AppendLogs(context.Context, *AppendRequest) (*AppendReply, error)
}

// UnimplementedRaftServer can be embedded to have forward compatible implementations.
type UnimplementedRaftServer struct {
}

func (*UnimplementedRaftServer) RequestVote(context.Context, *VoteRequest) (*VoteReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method RequestVote not implemented")
}
func (*UnimplementedRaftServer) AppendLogs(context.Context, *AppendRequest) (*AppendReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method AppendLogs not implemented")
}

func RegisterRaftServer(s *grpc.Server, srv RaftServer) {
	s.RegisterService(&_Raft_serviceDesc, srv)
}

func _Raft_RequestVote_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(VoteRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RaftServer).RequestVote(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/raft.Raft/RequestVote",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RaftServer).RequestVote(ctx, req.(*VoteRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Raft_AppendLogs_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(AppendRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RaftServer).AppendLogs(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/raft.Raft/AppendLogs",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RaftServer).AppendLogs(ctx, req.(*AppendRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var _Raft_serviceDesc = grpc.ServiceDesc{
	ServiceName: "raft.Raft",
	HandlerType: (*RaftServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "RequestVote",
			Handler:    _Raft_RequestVote_Handler,
		},
		{
			MethodName: "AppendLogs",
			Handler:    _Raft_AppendLogs_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "raft.proto",
}
