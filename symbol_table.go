package main

import "fmt"

type Scope string

const (
	FunctionScope Scope = "FunctionScope"
	ClassScope          = "ClassScope"
)

type SymbolTable struct {
	classScopeTable    map[string]Symbol
	functionScopeTable map[string]Symbol
}

func NewSymbolTable() SymbolTable {
	return SymbolTable{
		classScopeTable:    make(map[string]Symbol),
		functionScopeTable: make(map[string]Symbol),
	}
}

func nextIndex(table *map[string]Symbol, symbolType SymbolType) (index MachineWord) {
	for _, symbol := range *table {
		if symbol.symbolType == symbolType {
			index += 1
		}
	}
	return
}

func registerSymbol(table *map[string]Symbol, name string, symbol Symbol) Symbol {
	symbol.index = nextIndex(table, symbol.symbolType)
	(*table)[name] = symbol
	return symbol
}

func (s *SymbolTable) Count(symbolType SymbolType, scope Scope) (index MachineWord) {
	switch scope {
	case ClassScope:
		index = nextIndex(&s.classScopeTable, symbolType)
	case FunctionScope:
		index = nextIndex(&s.functionScopeTable, symbolType)
	}
	return
}

func (s *SymbolTable) Declare(symbol Symbol, name string, scope Scope) Symbol {
	switch scope {
	case ClassScope:
		symbol = registerSymbol(&s.classScopeTable, name, symbol)
	case FunctionScope:
		symbol = registerSymbol(&s.functionScopeTable, name, symbol)
	}
	fmt.Printf("Registered symbol %q: %q\n", name, symbol)
	return symbol
}

func (s *SymbolTable) Lookup(name string) (Symbol, error) {
	// Try to find it in the method scope table
	if symbol, ok := s.functionScopeTable[name]; ok {
		return symbol, nil
	}
	// Try to find it in the class scope table
	if symbol, ok := s.classScopeTable[name]; ok {
		return symbol, nil
	}
	// error
	return Symbol{}, fmt.Errorf("no symbol with name %q declared", name)
}

func (s *SymbolTable) Clear(scope Scope) {
	switch scope {
	case FunctionScope:
		s.functionScopeTable = make(map[string]Symbol)
		fallthrough
	case ClassScope:
		s.classScopeTable = make(map[string]Symbol)
	}
}
