package main

type SymbolType string

const (
	StaticSymbol   SymbolType = "static"
	FieldSymbol               = "field"
	ArgumentSymbol            = "argument"
	VarSymbol                 = "var"
	InvalidSymbol             = ""
)

type Symbol struct {
	symbolType   SymbolType
	variableType string
	index        MachineWord
}
