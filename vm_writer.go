package main

import (
	"fmt"
	"io"
	"strconv"
)

type VMSegmentType string

const (
	InvalidVMSegmentType VMSegmentType = ""
	ConstVMSegment       VMSegmentType = "constant"
	ArgumentVMSegment    VMSegmentType = "argument"
	LocalVMSegment       VMSegmentType = "local"
	StaticVMSegment      VMSegmentType = "static"
	ThisVMSegment        VMSegmentType = "this"
	ThatVMSegment        VMSegmentType = "that"
	PointerVMSegment     VMSegmentType = "pointer"
	TempVMSegment        VMSegmentType = "temp"
)

type VMOperation string

const (
	InvalidVMOperation VMOperation = ""
	AddVMOperation     VMOperation = "add"
	SubVMOperation     VMOperation = "sub"
	NegVMOperation     VMOperation = "neg"
	EqVMOperation      VMOperation = "eq"
	GtVMOperation      VMOperation = "gt"
	LtVMOperation      VMOperation = "lt"
	AndVMOperation     VMOperation = "and"
	OrvMOperation      VMOperation = "or"
	NotVMOperation     VMOperation = "not"
	MulVMOperation     VMOperation = "mul"
	DivVMOperation     VMOperation = "div"
)

type VMWriter struct {
	output io.Writer
}

func NewVMWriter(w io.Writer) VMWriter {
	return VMWriter{output: w}
}

func (w *VMWriter) WriteCommand(command string) {
	io.WriteString(w.output, command)
	io.WriteString(w.output, "\n")
}

func (w *VMWriter) WritePush(segment VMSegmentType, index MachineWord) {
	w.WriteCommand(fmt.Sprintf("push %s %d", segment, index))
}

func (w *VMWriter) WritePop(segment VMSegmentType, index MachineWord) {
	w.WriteCommand(fmt.Sprintf("pop %s %d", segment, index))
}

func (w *VMWriter) WriteStringConstant(constant string) {
	w.WritePush(ConstVMSegment, MachineWord(len(constant)))
	w.WriteCall("String.new", 1)
	// Store allocated string pointer in temp segment
	w.WritePop(TempVMSegment, 0)
	for _, c := range constant {
		// Push stored pointer to object
		w.WritePush(TempVMSegment, 0)
		// Push the character
		w.WritePush(ConstVMSegment, MachineWord(c))
		// Append another character
		w.WriteCall("String.appendChar", 2)
		// Remove 0 return value
		w.WritePop(TempVMSegment, 1)
	}
	// Leave pointer to string constant on top of stack
	w.WritePush(TempVMSegment, 0)
}

func (w *VMWriter) WriteArithmetic(operation VMOperation) {
	switch operation {
	case DivVMOperation:
		w.WriteCall("Math.divide", 2)
	case MulVMOperation:
		w.WriteCall("Math.multiply", 2)
	default:
		w.WriteCommand(string(operation))
	}
}

func (w *VMWriter) WriteLabel(label string) {
	w.WriteCommand("label " + label)
}

func (w *VMWriter) WriteGoto(label string) {
	w.WriteCommand("goto " + label)
}

func (w *VMWriter) WriteIf(label string) {
	w.WriteCommand("if-goto " + label)
}

func (w *VMWriter) WriteCall(label string, nargs MachineWord) {
	w.WriteCommand("call " + label + " " + strconv.FormatUint(uint64(nargs), 10))
}

func (w *VMWriter) WriteFunction(label string, nlocals MachineWord) {
	w.WriteCommand("function " + label + " " + strconv.FormatUint(uint64(nlocals), 10))
}

func (w *VMWriter) WriteReturn() {
	w.WriteCommand("return")
}

func (w *VMWriter) Close() {
}
