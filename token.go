package main

import (
	"fmt"
	"strconv"
)

type MachineWord int16

type TokenType string

const (
	InvalidToken    TokenType = ""
	Keyword         TokenType = "keyword"
	SymbolTokenType TokenType = "symbol"
	IntegerConstant TokenType = "integerConstant"
	StringConstant  TokenType = "stringConstant"
	Identifier      TokenType = "identifier"
)

type Token struct {
	tokenType TokenType
	terminal  string
}

func IsTokenType(t Token, tt TokenType) bool {
	return t.tokenType == tt
}

func IsTerminal(t Token, terminals ...string) bool {
	for _, terminal := range terminals {
		if t.terminal == terminal {
			return true
		}
	}
	return false
}

func (t *Token) asInt() MachineWord {
	word, err := strconv.Atoi(t.terminal)
	// < 0 as - is an operator
	if err != nil || word > 32767 || word < 0 {
		fmt.Printf("Cannot parse %q as 16 bit int!", t)
		return MachineWord(0)
	}
	return MachineWord(word)
}
