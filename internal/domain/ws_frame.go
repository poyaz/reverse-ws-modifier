package domain

import (
	"bytes"
	"encoding/binary"
)

type OpcodeType byte

const (
	ContinuationOpcode OpcodeType = 0
	TextOpcode         OpcodeType = 1
	BinaryOpcode       OpcodeType = 2
	CloseOpcode        OpcodeType = 8
	PingOpcode         OpcodeType = 9
	PongOpcode         OpcodeType = 10
)

type Frame struct {
	IsFragment bool // if the
	Opcode     OpcodeType
	Reserved   byte
	IsMasked   bool
	Length     uint64
	Payload    []byte
}

// Pong Get the Pong Frame
func (f Frame) Pong() Frame {
	f.Opcode = 10
	return f
}

// Text Get Text Payload
func (f Frame) Text() string {
	return string(f.Payload)
}

// IsControl checks if the Frame is a control Frame identified by opcodes where the most significant bit of the opcode is 1
func (f *Frame) IsControl() bool {
	return f.Opcode&0x08 == 0x08
}

func (f *Frame) HasReservedOpcode() bool {
	return f.Opcode > 10 || (f.Opcode >= 3 && f.Opcode <= 7)
}

func (f *Frame) CloseCode() uint16 {
	var code uint16
	binary.Read(bytes.NewReader(f.Payload), binary.BigEndian, &code)
	return code
}
