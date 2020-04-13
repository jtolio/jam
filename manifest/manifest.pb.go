// Code generated by protoc-gen-go. DO NOT EDIT.
// source: manifest.proto

package manifest

import (
	fmt "fmt"
	proto "github.com/golang/protobuf/proto"
	timestamp "github.com/golang/protobuf/ptypes/timestamp"
	math "math"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion3 // please upgrade the proto package

type Metadata_Type int32

const (
	Metadata_UNKNOWN Metadata_Type = 0
	Metadata_FILE    Metadata_Type = 1
	Metadata_SYMLINK Metadata_Type = 2
)

var Metadata_Type_name = map[int32]string{
	0: "UNKNOWN",
	1: "FILE",
	2: "SYMLINK",
}

var Metadata_Type_value = map[string]int32{
	"UNKNOWN": 0,
	"FILE":    1,
	"SYMLINK": 2,
}

func (x Metadata_Type) String() string {
	return proto.EnumName(Metadata_Type_name, int32(x))
}

func (Metadata_Type) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_0bb23f43f7afb4c1, []int{2, 0}
}

type Range struct {
	Blob                 string   `protobuf:"bytes,1,opt,name=blob,proto3" json:"blob,omitempty"`
	Offset               int64    `protobuf:"varint,2,opt,name=offset,proto3" json:"offset,omitempty"`
	Length               int64    `protobuf:"varint,3,opt,name=length,proto3" json:"length,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *Range) Reset()         { *m = Range{} }
func (m *Range) String() string { return proto.CompactTextString(m) }
func (*Range) ProtoMessage()    {}
func (*Range) Descriptor() ([]byte, []int) {
	return fileDescriptor_0bb23f43f7afb4c1, []int{0}
}

func (m *Range) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Range.Unmarshal(m, b)
}
func (m *Range) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Range.Marshal(b, m, deterministic)
}
func (m *Range) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Range.Merge(m, src)
}
func (m *Range) XXX_Size() int {
	return xxx_messageInfo_Range.Size(m)
}
func (m *Range) XXX_DiscardUnknown() {
	xxx_messageInfo_Range.DiscardUnknown(m)
}

var xxx_messageInfo_Range proto.InternalMessageInfo

func (m *Range) GetBlob() string {
	if m != nil {
		return m.Blob
	}
	return ""
}

func (m *Range) GetOffset() int64 {
	if m != nil {
		return m.Offset
	}
	return 0
}

func (m *Range) GetLength() int64 {
	if m != nil {
		return m.Length
	}
	return 0
}

type Stream struct {
	Ranges               []*Range `protobuf:"bytes,1,rep,name=ranges,proto3" json:"ranges,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *Stream) Reset()         { *m = Stream{} }
func (m *Stream) String() string { return proto.CompactTextString(m) }
func (*Stream) ProtoMessage()    {}
func (*Stream) Descriptor() ([]byte, []int) {
	return fileDescriptor_0bb23f43f7afb4c1, []int{1}
}

func (m *Stream) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Stream.Unmarshal(m, b)
}
func (m *Stream) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Stream.Marshal(b, m, deterministic)
}
func (m *Stream) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Stream.Merge(m, src)
}
func (m *Stream) XXX_Size() int {
	return xxx_messageInfo_Stream.Size(m)
}
func (m *Stream) XXX_DiscardUnknown() {
	xxx_messageInfo_Stream.DiscardUnknown(m)
}

var xxx_messageInfo_Stream proto.InternalMessageInfo

func (m *Stream) GetRanges() []*Range {
	if m != nil {
		return m.Ranges
	}
	return nil
}

type Metadata struct {
	Type                 Metadata_Type        `protobuf:"varint,1,opt,name=type,proto3,enum=manifest.Metadata_Type" json:"type,omitempty"`
	Creation             *timestamp.Timestamp `protobuf:"bytes,2,opt,name=creation,proto3" json:"creation,omitempty"`
	Modified             *timestamp.Timestamp `protobuf:"bytes,3,opt,name=modified,proto3" json:"modified,omitempty"`
	Mode                 uint32               `protobuf:"varint,4,opt,name=mode,proto3" json:"mode,omitempty"`
	LinkTarget           string               `protobuf:"bytes,5,opt,name=link_target,json=linkTarget,proto3" json:"link_target,omitempty"`
	XXX_NoUnkeyedLiteral struct{}             `json:"-"`
	XXX_unrecognized     []byte               `json:"-"`
	XXX_sizecache        int32                `json:"-"`
}

func (m *Metadata) Reset()         { *m = Metadata{} }
func (m *Metadata) String() string { return proto.CompactTextString(m) }
func (*Metadata) ProtoMessage()    {}
func (*Metadata) Descriptor() ([]byte, []int) {
	return fileDescriptor_0bb23f43f7afb4c1, []int{2}
}

func (m *Metadata) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Metadata.Unmarshal(m, b)
}
func (m *Metadata) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Metadata.Marshal(b, m, deterministic)
}
func (m *Metadata) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Metadata.Merge(m, src)
}
func (m *Metadata) XXX_Size() int {
	return xxx_messageInfo_Metadata.Size(m)
}
func (m *Metadata) XXX_DiscardUnknown() {
	xxx_messageInfo_Metadata.DiscardUnknown(m)
}

var xxx_messageInfo_Metadata proto.InternalMessageInfo

func (m *Metadata) GetType() Metadata_Type {
	if m != nil {
		return m.Type
	}
	return Metadata_UNKNOWN
}

func (m *Metadata) GetCreation() *timestamp.Timestamp {
	if m != nil {
		return m.Creation
	}
	return nil
}

func (m *Metadata) GetModified() *timestamp.Timestamp {
	if m != nil {
		return m.Modified
	}
	return nil
}

func (m *Metadata) GetMode() uint32 {
	if m != nil {
		return m.Mode
	}
	return 0
}

func (m *Metadata) GetLinkTarget() string {
	if m != nil {
		return m.LinkTarget
	}
	return ""
}

type Content struct {
	Metadata             *Metadata `protobuf:"bytes,1,opt,name=metadata,proto3" json:"metadata,omitempty"`
	Data                 *Stream   `protobuf:"bytes,2,opt,name=data,proto3" json:"data,omitempty"`
	Hash                 string    `protobuf:"bytes,3,opt,name=hash,proto3" json:"hash,omitempty"`
	XXX_NoUnkeyedLiteral struct{}  `json:"-"`
	XXX_unrecognized     []byte    `json:"-"`
	XXX_sizecache        int32     `json:"-"`
}

func (m *Content) Reset()         { *m = Content{} }
func (m *Content) String() string { return proto.CompactTextString(m) }
func (*Content) ProtoMessage()    {}
func (*Content) Descriptor() ([]byte, []int) {
	return fileDescriptor_0bb23f43f7afb4c1, []int{3}
}

func (m *Content) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Content.Unmarshal(m, b)
}
func (m *Content) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Content.Marshal(b, m, deterministic)
}
func (m *Content) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Content.Merge(m, src)
}
func (m *Content) XXX_Size() int {
	return xxx_messageInfo_Content.Size(m)
}
func (m *Content) XXX_DiscardUnknown() {
	xxx_messageInfo_Content.DiscardUnknown(m)
}

var xxx_messageInfo_Content proto.InternalMessageInfo

func (m *Content) GetMetadata() *Metadata {
	if m != nil {
		return m.Metadata
	}
	return nil
}

func (m *Content) GetData() *Stream {
	if m != nil {
		return m.Data
	}
	return nil
}

func (m *Content) GetHash() string {
	if m != nil {
		return m.Hash
	}
	return ""
}

type Entry struct {
	Path                 string   `protobuf:"bytes,1,opt,name=path,proto3" json:"path,omitempty"`
	Content              *Content `protobuf:"bytes,2,opt,name=content,proto3" json:"content,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *Entry) Reset()         { *m = Entry{} }
func (m *Entry) String() string { return proto.CompactTextString(m) }
func (*Entry) ProtoMessage()    {}
func (*Entry) Descriptor() ([]byte, []int) {
	return fileDescriptor_0bb23f43f7afb4c1, []int{4}
}

func (m *Entry) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Entry.Unmarshal(m, b)
}
func (m *Entry) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Entry.Marshal(b, m, deterministic)
}
func (m *Entry) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Entry.Merge(m, src)
}
func (m *Entry) XXX_Size() int {
	return xxx_messageInfo_Entry.Size(m)
}
func (m *Entry) XXX_DiscardUnknown() {
	xxx_messageInfo_Entry.DiscardUnknown(m)
}

var xxx_messageInfo_Entry proto.InternalMessageInfo

func (m *Entry) GetPath() string {
	if m != nil {
		return m.Path
	}
	return ""
}

func (m *Entry) GetContent() *Content {
	if m != nil {
		return m.Content
	}
	return nil
}

type EntrySet struct {
	Entries              []*Entry `protobuf:"bytes,1,rep,name=entries,proto3" json:"entries,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *EntrySet) Reset()         { *m = EntrySet{} }
func (m *EntrySet) String() string { return proto.CompactTextString(m) }
func (*EntrySet) ProtoMessage()    {}
func (*EntrySet) Descriptor() ([]byte, []int) {
	return fileDescriptor_0bb23f43f7afb4c1, []int{5}
}

func (m *EntrySet) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_EntrySet.Unmarshal(m, b)
}
func (m *EntrySet) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_EntrySet.Marshal(b, m, deterministic)
}
func (m *EntrySet) XXX_Merge(src proto.Message) {
	xxx_messageInfo_EntrySet.Merge(m, src)
}
func (m *EntrySet) XXX_Size() int {
	return xxx_messageInfo_EntrySet.Size(m)
}
func (m *EntrySet) XXX_DiscardUnknown() {
	xxx_messageInfo_EntrySet.DiscardUnknown(m)
}

var xxx_messageInfo_EntrySet proto.InternalMessageInfo

func (m *EntrySet) GetEntries() []*Entry {
	if m != nil {
		return m.Entries
	}
	return nil
}

type Page struct {
	SortKey string `protobuf:"bytes,1,opt,name=sort_key,json=sortKey,proto3" json:"sort_key,omitempty"`
	// Types that are valid to be assigned to Descendents:
	//	*Page_Branch
	//	*Page_Entries
	Descendents          isPage_Descendents `protobuf_oneof:"descendents"`
	XXX_NoUnkeyedLiteral struct{}           `json:"-"`
	XXX_unrecognized     []byte             `json:"-"`
	XXX_sizecache        int32              `json:"-"`
}

func (m *Page) Reset()         { *m = Page{} }
func (m *Page) String() string { return proto.CompactTextString(m) }
func (*Page) ProtoMessage()    {}
func (*Page) Descriptor() ([]byte, []int) {
	return fileDescriptor_0bb23f43f7afb4c1, []int{6}
}

func (m *Page) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Page.Unmarshal(m, b)
}
func (m *Page) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Page.Marshal(b, m, deterministic)
}
func (m *Page) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Page.Merge(m, src)
}
func (m *Page) XXX_Size() int {
	return xxx_messageInfo_Page.Size(m)
}
func (m *Page) XXX_DiscardUnknown() {
	xxx_messageInfo_Page.DiscardUnknown(m)
}

var xxx_messageInfo_Page proto.InternalMessageInfo

func (m *Page) GetSortKey() string {
	if m != nil {
		return m.SortKey
	}
	return ""
}

type isPage_Descendents interface {
	isPage_Descendents()
}

type Page_Branch struct {
	Branch *Stream `protobuf:"bytes,2,opt,name=branch,proto3,oneof"`
}

type Page_Entries struct {
	Entries *EntrySet `protobuf:"bytes,3,opt,name=entries,proto3,oneof"`
}

func (*Page_Branch) isPage_Descendents() {}

func (*Page_Entries) isPage_Descendents() {}

func (m *Page) GetDescendents() isPage_Descendents {
	if m != nil {
		return m.Descendents
	}
	return nil
}

func (m *Page) GetBranch() *Stream {
	if x, ok := m.GetDescendents().(*Page_Branch); ok {
		return x.Branch
	}
	return nil
}

func (m *Page) GetEntries() *EntrySet {
	if x, ok := m.GetDescendents().(*Page_Entries); ok {
		return x.Entries
	}
	return nil
}

// XXX_OneofWrappers is for the internal use of the proto package.
func (*Page) XXX_OneofWrappers() []interface{} {
	return []interface{}{
		(*Page_Branch)(nil),
		(*Page_Entries)(nil),
	}
}

type HashedData struct {
	Hash                 string   `protobuf:"bytes,1,opt,name=hash,proto3" json:"hash,omitempty"`
	Data                 *Stream  `protobuf:"bytes,2,opt,name=data,proto3" json:"data,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *HashedData) Reset()         { *m = HashedData{} }
func (m *HashedData) String() string { return proto.CompactTextString(m) }
func (*HashedData) ProtoMessage()    {}
func (*HashedData) Descriptor() ([]byte, []int) {
	return fileDescriptor_0bb23f43f7afb4c1, []int{7}
}

func (m *HashedData) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_HashedData.Unmarshal(m, b)
}
func (m *HashedData) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_HashedData.Marshal(b, m, deterministic)
}
func (m *HashedData) XXX_Merge(src proto.Message) {
	xxx_messageInfo_HashedData.Merge(m, src)
}
func (m *HashedData) XXX_Size() int {
	return xxx_messageInfo_HashedData.Size(m)
}
func (m *HashedData) XXX_DiscardUnknown() {
	xxx_messageInfo_HashedData.DiscardUnknown(m)
}

var xxx_messageInfo_HashedData proto.InternalMessageInfo

func (m *HashedData) GetHash() string {
	if m != nil {
		return m.Hash
	}
	return ""
}

func (m *HashedData) GetData() *Stream {
	if m != nil {
		return m.Data
	}
	return nil
}

type HashSet struct {
	Hashes               []*HashedData `protobuf:"bytes,1,rep,name=hashes,proto3" json:"hashes,omitempty"`
	XXX_NoUnkeyedLiteral struct{}      `json:"-"`
	XXX_unrecognized     []byte        `json:"-"`
	XXX_sizecache        int32         `json:"-"`
}

func (m *HashSet) Reset()         { *m = HashSet{} }
func (m *HashSet) String() string { return proto.CompactTextString(m) }
func (*HashSet) ProtoMessage()    {}
func (*HashSet) Descriptor() ([]byte, []int) {
	return fileDescriptor_0bb23f43f7afb4c1, []int{8}
}

func (m *HashSet) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_HashSet.Unmarshal(m, b)
}
func (m *HashSet) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_HashSet.Marshal(b, m, deterministic)
}
func (m *HashSet) XXX_Merge(src proto.Message) {
	xxx_messageInfo_HashSet.Merge(m, src)
}
func (m *HashSet) XXX_Size() int {
	return xxx_messageInfo_HashSet.Size(m)
}
func (m *HashSet) XXX_DiscardUnknown() {
	xxx_messageInfo_HashSet.DiscardUnknown(m)
}

var xxx_messageInfo_HashSet proto.InternalMessageInfo

func (m *HashSet) GetHashes() []*HashedData {
	if m != nil {
		return m.Hashes
	}
	return nil
}

func init() {
	proto.RegisterEnum("manifest.Metadata_Type", Metadata_Type_name, Metadata_Type_value)
	proto.RegisterType((*Range)(nil), "manifest.Range")
	proto.RegisterType((*Stream)(nil), "manifest.Stream")
	proto.RegisterType((*Metadata)(nil), "manifest.Metadata")
	proto.RegisterType((*Content)(nil), "manifest.Content")
	proto.RegisterType((*Entry)(nil), "manifest.Entry")
	proto.RegisterType((*EntrySet)(nil), "manifest.EntrySet")
	proto.RegisterType((*Page)(nil), "manifest.Page")
	proto.RegisterType((*HashedData)(nil), "manifest.HashedData")
	proto.RegisterType((*HashSet)(nil), "manifest.HashSet")
}

func init() {
	proto.RegisterFile("manifest.proto", fileDescriptor_0bb23f43f7afb4c1)
}

var fileDescriptor_0bb23f43f7afb4c1 = []byte{
	// 522 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x8c, 0x53, 0x4d, 0x6f, 0xd3, 0x40,
	0x10, 0xad, 0x13, 0xc7, 0x76, 0xc7, 0x6a, 0x09, 0x2b, 0x04, 0xa6, 0x97, 0x46, 0x16, 0x12, 0xa1,
	0x45, 0xae, 0x08, 0x02, 0xee, 0x85, 0x56, 0xa9, 0xd2, 0x06, 0xb4, 0x09, 0x42, 0x70, 0xa9, 0x36,
	0xf1, 0xc4, 0xb1, 0x1a, 0xef, 0x46, 0xf6, 0x70, 0xc8, 0x3f, 0xe0, 0xc0, 0x8f, 0x46, 0xbb, 0xfe,
	0x42, 0x7c, 0x08, 0x6e, 0x3b, 0x33, 0x6f, 0x66, 0xde, 0x7b, 0x63, 0xc3, 0x61, 0x26, 0x64, 0xba,
	0xc2, 0x82, 0xa2, 0x6d, 0xae, 0x48, 0x31, 0xaf, 0x8e, 0x8f, 0x8e, 0x13, 0xa5, 0x92, 0x0d, 0x9e,
	0x99, 0xfc, 0xe2, 0xeb, 0xea, 0x8c, 0xd2, 0x0c, 0x0b, 0x12, 0xd9, 0xb6, 0x84, 0x86, 0x13, 0xe8,
	0x71, 0x21, 0x13, 0x64, 0x0c, 0xec, 0xc5, 0x46, 0x2d, 0x02, 0x6b, 0x60, 0x0d, 0xf7, 0xb9, 0x79,
	0xb3, 0x87, 0xe0, 0xa8, 0xd5, 0xaa, 0x40, 0x0a, 0x3a, 0x03, 0x6b, 0xd8, 0xe5, 0x55, 0xa4, 0xf3,
	0x1b, 0x94, 0x09, 0xad, 0x83, 0x6e, 0x99, 0x2f, 0xa3, 0xf0, 0x05, 0x38, 0x33, 0xca, 0x51, 0x64,
	0xec, 0x29, 0x38, 0xb9, 0x1e, 0x5b, 0x04, 0xd6, 0xa0, 0x3b, 0xf4, 0x47, 0xf7, 0xa2, 0x86, 0xa2,
	0x59, 0xc7, 0xab, 0x72, 0xf8, 0xad, 0x03, 0xde, 0x0d, 0x92, 0x88, 0x05, 0x09, 0x76, 0x0a, 0x36,
	0xed, 0xb6, 0x68, 0x38, 0x1c, 0x8e, 0x1e, 0xb5, 0x3d, 0x35, 0x22, 0x9a, 0xef, 0xb6, 0xc8, 0x0d,
	0x88, 0xbd, 0x06, 0x6f, 0x99, 0xa3, 0xa0, 0x54, 0x49, 0x43, 0xcf, 0x1f, 0x1d, 0x45, 0xa5, 0xda,
	0xa8, 0x56, 0x1b, 0xcd, 0x6b, 0xb5, 0xbc, 0xc1, 0xea, 0xbe, 0x4c, 0xc5, 0xe9, 0x2a, 0xc5, 0xd8,
	0xd0, 0xff, 0x47, 0x5f, 0x8d, 0xd5, 0x06, 0x65, 0x2a, 0xc6, 0xc0, 0x1e, 0x58, 0xc3, 0x03, 0x6e,
	0xde, 0xec, 0x18, 0xfc, 0x4d, 0x2a, 0xef, 0x6e, 0x49, 0xe4, 0x09, 0x52, 0xd0, 0x33, 0xde, 0x81,
	0x4e, 0xcd, 0x4d, 0x26, 0x3c, 0x01, 0x5b, 0x53, 0x66, 0x3e, 0xb8, 0x1f, 0xa7, 0x93, 0xe9, 0xfb,
	0x4f, 0xd3, 0xfe, 0x1e, 0xf3, 0xc0, 0xbe, 0xbc, 0xba, 0xbe, 0xe8, 0x5b, 0x3a, 0x3d, 0xfb, 0x7c,
	0x73, 0x7d, 0x35, 0x9d, 0xf4, 0x3b, 0x61, 0x01, 0xee, 0x5b, 0x25, 0x09, 0x25, 0xb1, 0x08, 0xbc,
	0xac, 0x92, 0x6c, 0xcc, 0xf0, 0x47, 0xec, 0x77, 0x33, 0x78, 0x83, 0x61, 0x4f, 0xc0, 0x36, 0xd8,
	0xd2, 0x87, 0x7e, 0x8b, 0x2d, 0xcf, 0xc1, 0x4d, 0x55, 0x2b, 0x58, 0x8b, 0xa2, 0x3c, 0xda, 0x3e,
	0x37, 0xef, 0x70, 0x0c, 0xbd, 0x0b, 0x49, 0xf9, 0x4e, 0x17, 0xb7, 0x82, 0xd6, 0xf5, 0xfd, 0xf5,
	0x9b, 0x9d, 0x82, 0xbb, 0x2c, 0x19, 0x55, 0x93, 0xef, 0xb7, 0x93, 0x2b, 0xaa, 0xbc, 0x46, 0x84,
	0xaf, 0xc0, 0x33, 0x93, 0x66, 0x48, 0xec, 0x19, 0xb8, 0x28, 0x29, 0x4f, 0xff, 0x74, 0x7f, 0x03,
	0xe2, 0x75, 0x3d, 0xfc, 0x6e, 0x81, 0xfd, 0x41, 0x24, 0xc8, 0x1e, 0x83, 0x57, 0xa8, 0x9c, 0x6e,
	0xef, 0x70, 0x57, 0x91, 0x70, 0x75, 0x3c, 0xc1, 0x1d, 0x3b, 0x01, 0x67, 0x91, 0x0b, 0xb9, 0x5c,
	0xff, 0x4d, 0xe0, 0x78, 0x8f, 0x57, 0x08, 0x16, 0xb5, 0xab, 0xbb, 0xbf, 0x3a, 0x57, 0xf3, 0x1b,
	0xef, 0x35, 0xfb, 0xcf, 0x0f, 0xc0, 0x8f, 0xb1, 0x58, 0xa2, 0x8c, 0x51, 0x52, 0x11, 0x5e, 0x02,
	0x8c, 0x45, 0xb1, 0xc6, 0xf8, 0xdd, 0xcf, 0x8e, 0x59, 0xad, 0x63, 0xff, 0xe7, 0x75, 0xf8, 0x06,
	0x5c, 0x3d, 0x47, 0x9b, 0xf1, 0x1c, 0x1c, 0xdd, 0xd8, 0x78, 0xf1, 0xa0, 0x6d, 0x69, 0x57, 0xf1,
	0x0a, 0x73, 0x0e, 0x5f, 0x9a, 0xbf, 0x77, 0xe1, 0x98, 0x0f, 0xf2, 0xe5, 0x8f, 0x00, 0x00, 0x00,
	0xff, 0xff, 0xe9, 0xcb, 0xa7, 0xf6, 0xe0, 0x03, 0x00, 0x00,
}
