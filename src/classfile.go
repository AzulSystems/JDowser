// Copyright 2020 Azul Systems, Inc. All rights reserved.
// Use of this source code is governed by the 3-Clause BSD
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"log"
)

type ConstantPool []ConstantPoolEntry

const (
	CONSTANT_Utf8               = 1
	CONSTANT_Integer            = 3
	CONSTANT_Float              = 4
	CONSTANT_Long               = 5
	CONSTANT_Double             = 6
	CONSTANT_Class              = 7
	CONSTANT_String             = 8
	CONSTANT_Fieldref           = 9
	CONSTANT_Methodref          = 10
	CONSTANT_InterfaceMethodref = 11
	CONSTANT_NameAndType        = 12
	CONSTANT_MethodHandle       = 15
	CONSTANT_MethodType         = 16
	CONSTANT_InvokeDynamic      = 18
)

type ClassFile struct {
	size         int
	magic        uint32
	minorVersion uint16
	majorVersion uint16
	constantPool ConstantPool
	accessFlags  uint16
	thisClass    uint16
	superClass   uint16
	interfaces   []uint16
	fields       []Field
	methods      []Method
	attributes   []Attribute
}

type ConstantPoolEntry interface {
	Read(reader *ClassFileReader) int
}

type Attribute interface {
	Read(reader *ClassFileReader) int
}

type Field struct {
	accessFlags     uint16
	nameIndex       uint16
	descriptorIndex uint16
	attributes      []Attribute
}

type Method struct {
	accessFlags     uint16
	nameIndex       uint16
	descriptorIndex uint16
	attributes      []Attribute
}

func ParseClassFile(bytes []byte) *ClassFile {
	reader := CreateClassFileReader(bytes)
	cf := &ClassFile{}
	cf.size = reader.Size()
	cf.magic = reader.ReadUint32()

	if cf.magic != 0xcafebabe {
		log.Fatal("Error reading class - not valid type")
	}

	cf.minorVersion = reader.ReadUint16()
	cf.majorVersion = reader.ReadUint16()
	cf.constantPool = readConstantPool(reader)
	cf.accessFlags = reader.ReadUint16()
	cf.thisClass = reader.ReadUint16()
	cf.superClass = reader.ReadUint16()
	cf.readInterfaces(reader)
	cf.fields = readFields(reader, cf.constantPool)
	cf.methods = readMethods(reader, cf.constantPool)
	cf.attributes = readAttributes(reader, cf.constantPool)
	return cf
}

func readConstantPool(r *ClassFileReader) ConstantPool {
	constantPool := make(ConstantPool, r.ReadUint16())
	for slot := 1; slot < len(constantPool); {
		cpInfo := newConstantPoolEntry(r.ReadUint8())
		slots := cpInfo.Read(r)
		constantPool[slot] = cpInfo
		slot += slots
	}
	return constantPool
}

func (cf *ClassFile) readInterfaces(r *ClassFileReader) {
	cf.interfaces = make([]uint16, r.ReadUint16())
	for i := 0; i < len(cf.interfaces); i++ {
		cf.interfaces[i] = r.ReadUint16()
	}
}

func readFields(reader *ClassFileReader, cp ConstantPool) []Field {
	fields := make([]Field, reader.ReadUint16())
	for i := 0; i < len(fields); i++ {
		fields[i] = Field{
			reader.ReadUint16(),
			reader.ReadUint16(),
			reader.ReadUint16(),
			readAttributes(reader, cp)}
	}
	return fields
}

func readMethods(reader *ClassFileReader, cp ConstantPool) []Method {
	methods := make([]Method, reader.ReadUint16())
	for i := 0; i < len(methods); i++ {
		methods[i] = Method{
			reader.ReadUint16(),
			reader.ReadUint16(),
			reader.ReadUint16(),
			readAttributes(reader, cp)}
	}
	return methods
}

func readAttributes(reader *ClassFileReader, cp ConstantPool) []Attribute {
	attributes := make([]Attribute, reader.ReadUint16())
	for i := 0; i < len(attributes); i++ {
		attributes[i] = readAttribute(reader, cp)
	}
	return attributes
}

func readAttribute(r *ClassFileReader, cp ConstantPool) Attribute {
	attrNameIndex := r.ReadUint16()
	attrLength := r.ReadUint32()
	if c, ok := cp.GetConstantEntry(attrNameIndex).(*ConstantUtf8Entry); ok {
		var attr Attribute
		switch attrName := c.String(); attrName {
		case "ConstantValue":
			attr = &ConstantValueAttribute{}
		default:
			r.ReadBytes(int(attrLength))
		}
		if attr != nil {
			attr.Read(r)
			return attr
		}
	}
	return nil
}

func (p ConstantPool) GetConstantEntry(index uint16) ConstantPoolEntry {
	return p[index]
}

func newConstantPoolEntry(constType uint8) ConstantPoolEntry {
	switch constType {
	case CONSTANT_Class:
		return &ConstantClassEntry{}
	case CONSTANT_Fieldref:
		return &ConstantFieldRefEntry{}
	case CONSTANT_Methodref:
		return &ConstantMethodRefEntry{}
	case CONSTANT_InterfaceMethodref:
		return &ConstantInterfaceMethodRefEntry{}
	case CONSTANT_String:
		return &ConstantStringEntry{}
	case CONSTANT_Integer:
		return &ConstantIntegerEntry{}
	case CONSTANT_Float:
		return &ConstantFloatEntry{}
	case CONSTANT_Long:
		return &ConstantLongEntry{}
	case CONSTANT_Double:
		return &ConstantDoubleEntry{}
	case CONSTANT_NameAndType:
		return &ConstantNameAndTypeEntry{}
	case CONSTANT_Utf8:
		return &ConstantUtf8Entry{}
	case CONSTANT_MethodHandle:
		return &ConstantMethodHandleEntry{}
	case CONSTANT_MethodType:
		return &ConstantMethodTypeEntry{}
	case CONSTANT_InvokeDynamic:
		return &ConstantInvokeDynamicEntry{}
	default:
		panic("Invalid const type: ")
	}
}

type ConstantValueAttribute struct {
	value uint16
}

type ConstantFieldRefEntry struct {
	class_idx     uint16
	name_type_idx uint16
}

type ConstantMethodRefEntry struct {
	class_idx     uint16
	name_type_idx uint16
}

type ConstantClassEntry struct {
	name_idx uint16
}

type ConstantInterfaceMethodRefEntry struct {
	class_idx     uint16
	name_type_idx uint16
}

type ConstantStringEntry struct {
	stringIndex uint16
}

type ConstantIntegerEntry struct {
	bytes uint32
}

type ConstantFloatEntry struct {
	bytes uint32
}

type ConstantLongEntry struct {
	highBytes uint32
	lowBytes  uint32
}

type ConstantDoubleEntry struct {
	highBytes uint32
	lowBytes  uint32
}

type ConstantNameAndTypeEntry struct {
	nameIdx       uint16
	descriptorIdx uint16
}

type ConstantUtf8Entry struct {
	bytes []byte
}

type ConstantMethodHandleEntry struct {
	referenceKind uint8
	referenceIdx  uint16
}
type ConstantMethodTypeEntry struct {
	descriptorIdx uint16
}

type ConstantInvokeDynamicEntry struct {
	bs_method_attr_idx uint16
	name_type_idx      uint16
}

func (e *ConstantValueAttribute) Read(reader *ClassFileReader) int {
	e.value = reader.ReadUint16()
	return 1
}

func (e *ConstantClassEntry) Read(r *ClassFileReader) int {
	e.name_idx = r.ReadUint16()
	return 1
}

func (e *ConstantFieldRefEntry) Read(r *ClassFileReader) int {
	e.class_idx = r.ReadUint16()
	e.name_type_idx = r.ReadUint16()
	return 1
}

func (e *ConstantMethodRefEntry) Read(r *ClassFileReader) int {
	e.class_idx = r.ReadUint16()
	e.name_type_idx = r.ReadUint16()
	return 1
}

func (e *ConstantInterfaceMethodRefEntry) Read(r *ClassFileReader) int {
	e.class_idx = r.ReadUint16()
	e.name_type_idx = r.ReadUint16()
	return 1
}

func (e *ConstantStringEntry) Read(r *ClassFileReader) int {
	e.stringIndex = r.ReadUint16()
	return 1
}

func (e *ConstantStringEntry) Value(cp ConstantPool) interface{} {
	if val, ok := cp.GetConstantEntry(e.stringIndex).(*ConstantUtf8Entry); ok {
		return val.String()
	}
	return fmt.Sprintf("%v", cp.GetConstantEntry(e.stringIndex))
}

func (e *ConstantIntegerEntry) Read(r *ClassFileReader) int {
	e.bytes = r.ReadUint32()
	return 1
}

func (e *ConstantFloatEntry) Read(r *ClassFileReader) int {
	e.bytes = r.ReadUint32()
	return 1
}

func (e *ConstantLongEntry) Read(r *ClassFileReader) int {
	e.highBytes = r.ReadUint32()
	e.lowBytes = r.ReadUint32()
	return 2
}

func (e *ConstantDoubleEntry) Read(r *ClassFileReader) int {
	e.highBytes = r.ReadUint32()
	e.lowBytes = r.ReadUint32()
	return 2
}

func (e *ConstantNameAndTypeEntry) Read(r *ClassFileReader) int {
	e.nameIdx = r.ReadUint16()
	e.descriptorIdx = r.ReadUint16()
	return 1
}

func (e *ConstantUtf8Entry) Read(r *ClassFileReader) int {
	e.bytes = r.ReadBytes(int(r.ReadUint16()))
	return 1
}

func (e *ConstantUtf8Entry) String() string {
	return fmt.Sprintf("%s", e.bytes)
}

func (e *ConstantMethodHandleEntry) Read(r *ClassFileReader) int {
	e.referenceKind = r.ReadBytes(1)[0]
	e.referenceIdx = r.ReadUint16()
	return 1
}

func (e *ConstantMethodTypeEntry) Read(r *ClassFileReader) int {
	e.descriptorIdx = r.ReadUint16()
	return 1
}

func (e *ConstantInvokeDynamicEntry) Read(r *ClassFileReader) int {
	e.bs_method_attr_idx = r.ReadUint16()
	e.name_type_idx = r.ReadUint16()
	return 1
}
