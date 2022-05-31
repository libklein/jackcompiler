package main

import (
	"fmt"
	"strconv"
	"strings"
)

type SubroutineType string

const (
	InvalidSubroutineType     SubroutineType = ""
	MethodSubroutineType      SubroutineType = "method"
	FunctionSubroutineType    SubroutineType = "function"
	ConstructorSubroutineType SubroutineType = "constructor"
)

type TokenScanner interface {
	Token() Token
	Err() error
	Scan() bool
}

type OutputWriter interface {
	WriteCommand(string)
	WritePush(VMSegmentType, MachineWord)
	WritePop(VMSegmentType, MachineWord)
	WriteArithmetic(operation VMOperation)
	WriteLabel(string)
	WriteGoto(string)
	WriteIf(string)
	WriteCall(string, MachineWord)
	WriteFunction(string, MachineWord)
	WriteStringConstant(string)
	WriteReturn()
}

type JackCompiler struct {
	tokenScanner     TokenScanner
	symbolTable      SymbolTable
	output           OutputWriter
	currentClassName string
	nextLabelID      uint64
}

func NewJackCompiler(tokenScanner TokenScanner, output OutputWriter) *JackCompiler {
	return &JackCompiler{
		tokenScanner: tokenScanner,
		symbolTable:  NewSymbolTable(),
		output:       output,
	}
}

func (c *JackCompiler) generateLabel() string {
	labelID := c.nextLabelID
	c.nextLabelID += 1
	return "L" + strconv.FormatUint(labelID, 10) + ":"
}

func (c *JackCompiler) writeFunction(functionName string, nargs MachineWord) {
	c.output.WriteFunction(c.currentClassName+"."+functionName, nargs)
}

func (c *JackCompiler) generateVariableAccess(varName string) (VMSegmentType, MachineWord) {
	symbol, err := c.symbolTable.Lookup(varName)
	if err != nil {
		panic(fmt.Sprintf("Unknown variable: %q\n", varName))
	}

	switch symbol.symbolType {
	case StaticSymbol:
		return StaticVMSegment, symbol.index
	case ArgumentSymbol:
		return ArgumentVMSegment, symbol.index
	case VarSymbol:
		return LocalVMSegment, symbol.index
	case FieldSymbol:
		return ThisVMSegment, symbol.index
	default:
		panic(fmt.Sprintf("Unknown symbolType: %q\n", symbol.symbolType))
	}
}

func (c *JackCompiler) generateArrayElemPointer(name string) {
	// Stores offset on top of stack
	c.compileExpression()

	// Emit code that moves the that pointer
	// Store base addr on stack
	segment, index := c.generateVariableAccess(name)
	c.output.WritePush(segment, index)
	// Add together
	c.output.WriteArithmetic(AddVMOperation)
}

func (c *JackCompiler) nextToken() Token {
	return c.tokenScanner.Token()
}

func (c *JackCompiler) advance() Token {
	if !c.tokenScanner.Scan() {
		panic("Could not advance token scanner!")
	}
	return c.nextToken()
}

func (c *JackCompiler) consume(expectedTerminals ...string) {
	if len(expectedTerminals) == 0 {
		c.advance()
		return
	}

	for _, expectedTerminal := range expectedTerminals {
		if !IsTerminal(c.nextToken(), expectedTerminal) {
			panic("Expected terminal \"" + expectedTerminal + "\", got \"" + c.nextToken().terminal + "\"")
		}
		c.advance()
	}
}

func (c *JackCompiler) Compile() {
	c.advance()
	c.compileClass()
	return
}

func (c *JackCompiler) compileClass() {
	c.consume("class")

	c.symbolTable.Clear(ClassScope)

	if className, err := parseIdentifier(c.nextToken()); err == nil {
		c.currentClassName = className
		c.advance()
	} else {
		panic(err)
	}

	c.consume("{")
	for c.compileClassVarDec() == nil {
	}
	for c.compileSubroutineDec() == nil {
	}
	// Return an error if the next terminal is not } or we are not at EOF
	if c.nextToken().terminal != "}" || c.tokenScanner.Scan() {
		panic("Unexpected end of class")
	}
}

func (c *JackCompiler) compileClassVarDec() error {
	switch token := c.nextToken(); {
	case IsTerminal(token, "static"):
		c.consume("static")
		c.compileVarSequence(StaticSymbol, ClassScope)
	case IsTerminal(token, "field"):
		c.consume("field")
		c.compileVarSequence(FieldSymbol, ClassScope)
	default:
		return fmt.Errorf("Expected \"static\" or \"field\" but got %s", token.terminal)
	}
	return nil
}

func (c *JackCompiler) compileVarSequence(symbolType SymbolType, symbolScope Scope) (numDeclarations MachineWord) {
	symbol := Symbol{symbolType: symbolType}

	symbol.variableType, _ = parseType(c.nextToken())
	c.consume()

	for {
		varName, _ := parseIdentifier(c.nextToken())
		c.consume() // consume identifier

		numDeclarations += 1

		// Register types in symbol table
		c.symbolTable.Declare(symbol, varName, symbolScope)
		if IsTerminal(c.nextToken(), ",") {
			c.consume(",")
		} else {
			break
		}
	}
	c.consume(";")
	return numDeclarations
}

func (c *JackCompiler) compileSubroutineDec() error {
	c.symbolTable.Clear(FunctionScope)

	methodType, err := parseSubroutineType(c.nextToken())
	if err != nil {
		return err
	}

	if methodType == MethodSubroutineType {
		// Method will get an extra argument not captured in the parameter list.
		thisSymbol := Symbol{
			symbolType:   ArgumentSymbol,
			variableType: c.currentClassName,
		}

		c.symbolTable.Declare(thisSymbol, "this", FunctionScope)
	}

	c.consume()
	name, _ := parseIdentifier(c.advance())
	c.consume() // Consume identfier

	c.consume("(")

	if !IsTerminal(c.nextToken(), ")") {
		c.compileParameterList()
	}

	c.consume(")")

	c.compileSubroutine(name, methodType)

	return nil
}

func (c *JackCompiler) compileSubroutine(name string, subroutineType SubroutineType) {
	c.consume("{")
	nlocals := MachineWord(0)
	for {
		varCount := c.compileVarDec()
		if varCount == 0 {
			break
		}
		nlocals += varCount
	}

	c.writeFunction(name, nlocals)

	switch subroutineType {
	case ConstructorSubroutineType:
		// Get count of field variables
		nfields := c.symbolTable.Count(FieldSymbol, ClassScope)
		// Allocate this pointer
		c.output.WritePush(ConstVMSegment, nfields)
		c.output.WriteCall("Memory.alloc", 1)
		// Set THIS pointer
		c.output.WritePop(PointerVMSegment, 0)
	case MethodSubroutineType:
		// Write output
		c.output.WritePush(ArgumentVMSegment, 0)
		c.output.WritePop(PointerVMSegment, 0)
	}

	c.compileStatements()
	c.consume("}")
}

func (c *JackCompiler) compileParameterList() {
	symbol := Symbol{symbolType: ArgumentSymbol}
	for {
		symbol.variableType, _ = parseType(c.nextToken())
		c.consume()
		varName, _ := parseVarName(c.nextToken())
		c.consume()

		// Register types in symbol table
		c.symbolTable.Declare(symbol, varName, FunctionScope)

		if IsTerminal(c.nextToken(), ",") {
			c.consume(",")
		} else {
			break
		}
	}
}

func (c *JackCompiler) compileVarDec() MachineWord {
	if !IsTerminal(c.nextToken(), "var") {
		return 0
	}
	c.consume("var")
	return c.compileVarSequence(VarSymbol, FunctionScope)
}

func (c *JackCompiler) compileStatements() {
	for !IsTerminal(c.nextToken(), "}") {
		// Compile next statement
		switch token := c.nextToken(); {
		case IsTerminal(token, "let"):
			c.compileLet()
		case IsTerminal(token, "if"):
			c.compileIf()
		case IsTerminal(token, "while"):
			c.compileWhile()
		case IsTerminal(token, "do"):
			c.compileDo()
		case IsTerminal(token, "return"):
			c.compileReturn()
		default:
			panic("unexpected token " + token.terminal)
		}
	}
}

func (c *JackCompiler) compileDo() {
	c.consume("do")
	c.compileSubroutineCall("")

	// Discard unused return value
	c.output.WritePop(TempVMSegment, 0)

	c.consume(";")
}

func (c *JackCompiler) compileLet() {
	varName := c.advance().terminal
	// Where to store the result of the RHS expression
	isArrayAccess := false

	// Evaluate destination address if LHS is an array
	if IsTerminal(c.advance(), "[") {
		isArrayAccess = true
		c.consume("[")
		c.generateArrayElemPointer(varName)
		// Adress *varName + expr_result is not on top of stack
		c.consume("]")
	}

	// Handle RHS
	c.consume("=")
	c.compileExpression()
	c.consume(";")
	// Layout: Value of expression is on top of stack.
	//		   -> Pop last value into var Name
	if isArrayAccess {
		// Save result of RHS expression in temp
		c.output.WritePop(TempVMSegment, 0)
		// Pop array element address into pointer (THAT)
		c.output.WritePop(PointerVMSegment, 1)
		// Restore rhs rexpression result from temp
		c.output.WritePush(TempVMSegment, 0)
		// Pop into destination
		c.output.WritePop(ThatVMSegment, 0)
	} else {
		segment, index := c.generateVariableAccess(varName)
		c.output.WritePop(segment, index)
	}
}

func (c *JackCompiler) compileWhile() {
	c.consume("while", "(")

	nextLabelPrefix := c.generateLabel()

	c.output.WriteLabel(nextLabelPrefix + "BEGIN")

	if err := c.compileExpression(); err != nil {
		panic(err)
	}

	c.output.WriteArithmetic(NotVMOperation)
	c.output.WriteIf(nextLabelPrefix + "EXIT")

	c.consume(")", "{")

	c.compileStatements()
	c.consume("}")

	c.output.WriteGoto(nextLabelPrefix + "BEGIN")
	c.output.WriteLabel(nextLabelPrefix + "EXIT")
}

func (c *JackCompiler) compileReturn() {
	c.consume("return")
	// May have an expression, may not
	if c.compileExpression() != nil {
		// If not, push 0
		c.output.WritePush(ConstVMSegment, 0)
	}
	c.output.WriteReturn()
	// Otherwise the return value will already be on the stack
	c.consume(";")
}

func (c *JackCompiler) compileIf() {
	c.consume("if", "(")

	labelPrefix := c.generateLabel()

	if err := c.compileExpression(); err != nil {
		panic(err)
	}

	c.output.WriteArithmetic(NotVMOperation)
	c.output.WriteIf(labelPrefix + "ELSE")

	c.consume(")", "{")
	c.compileStatements()
	c.consume("}")

	c.output.WriteGoto(labelPrefix + "END")
	c.output.WriteLabel(labelPrefix + "ELSE")

	if IsTerminal(c.nextToken(), "else") {
		c.consume("else", "{")
		c.compileStatements()
		c.consume("}")
	}

	c.output.WriteLabel(labelPrefix + "END")
}

func (c *JackCompiler) compileExpression() error {
	if err := c.compileTerm(); err != nil {
		return err
	}
	token := c.nextToken()
	if isBinaryOp(token) {
		op := parseBinaryOp(token)
		c.advance()
		c.compileTerm()
		// Emit code
		c.output.WriteArithmetic(op)
	}
	return nil
}

/*
* Expression list: (expression (, expression)*)?
 */
func (c *JackCompiler) compileExpressionList() (i MachineWord) {
	for c.compileExpression() == nil {
		i += 1
		if !IsTerminal(c.nextToken(), ",") {
			break
		}
		c.consume(",")
	}
	return i
}

func (c *JackCompiler) compileSubroutineCall(name string) {
	/**
	* Examples:
	*	- do Memory.init();
	*		 ^ name ^ method name
	*	- do square.dispose();
	*		 ^ name ^ method name
	 */

	// Try to determine variable/function name if not given
	if name == "" {
		var err error
		name, err = parseIdentifier(c.nextToken())
		if err != nil {
			panic(err)
		}
		c.advance()
	}

	switch c.nextToken().terminal {
	case ".":
		c.consume(".")
		methodName, err := parseIdentifier(c.nextToken())
		if err != nil {
			panic(err)
		}
		// Advance over identifier
		c.advance()

		nargs := MachineWord(0)
		// Check if name is a symbol! If it is, push the object on the stack
		if symbol, err := c.symbolTable.Lookup(name); err == nil {
			nargs += 1 // Account for this pointer

			// Push the address of the object a method is called on onto the stack.
			// This will be argument 0 (this pointer)
			segment, index := c.generateVariableAccess(name)
			c.output.WritePush(segment, index)

			name = symbol.variableType + "." + methodName
		} else {
			// Name refers to some function. Needs to be fully qualified
			name = name + "." + methodName
		}

		c.consume("(")
		nargs += c.compileExpressionList()
		c.consume(")")

		c.output.WriteCall(name, nargs)
	case "(":
		// Push pointer of this object
		c.output.WritePush(PointerVMSegment, 0)
		// We call a local method. It is not allowed to call functions without prefixing the class name.
		c.consume("(")
		nargs := 1 + c.compileExpressionList()
		c.consume(")")
		c.output.WriteCall(c.currentClassName+"."+name, nargs)
	default:
		panic("Expected terminal ( or ., but got " + c.nextToken().terminal)
	}
}

func (c *JackCompiler) compileVarNameSubterm() error {
	// Parse var name
	varNameToken := c.nextToken()
	varName, err := parseVarName(varNameToken)
	if err != nil {
		return fmt.Errorf("unable to parse variable name %q", varNameToken.terminal)
	}
	c.advance()

	switch c.nextToken().terminal {
	case "[":
		c.consume("[")

		c.generateArrayElemPointer(varName)
		// Address *varName + expr_result is now on top of stack
		// Pop into pointer (THAT)
		c.output.WritePop(PointerVMSegment, 1)
		// Push value onto stack
		c.output.WritePush(ThatVMSegment, 0)

		c.consume("]")
	case "(", ".":
		c.compileSubroutineCall(varName)
	default:
		// Direct access to varName
		segment, index := c.generateVariableAccess(varName)
		c.output.WritePush(segment, index)
	}
	return nil
}

/*
 * Term:
 * integerConstant | stringConstant | keywordConstant | varName | varName '[' expression ']' |
 * subroutineCall | '(' expression ')' | unaryOp term*
 */
func (c *JackCompiler) compileTerm() error {
	switch token := c.nextToken(); {
	case IsTokenType(token, IntegerConstant):
		if constant, err := parseIntegerConstant(token); err == nil {
			c.output.WritePush(ConstVMSegment, constant)
			c.advance()
		} else {
			panic(err)
		}
		return nil
	case IsTokenType(token, StringConstant):
		c.output.WriteStringConstant(token.terminal)
		// Consume string constant
		c.advance()
		return nil
	case IsTokenType(token, Keyword):
		switch {
		case IsTerminal(token, "true"):
			c.output.WritePush(ConstVMSegment, 0)
			c.output.WriteArithmetic(NotVMOperation)
		case IsTerminal(token, "false"):
			c.output.WritePush(ConstVMSegment, 0)
		case IsTerminal(token, "null"):
			c.output.WritePush(ConstVMSegment, 0)
		case IsTerminal(token, "this"):
			// Push "this" pointer onto stack
			c.output.WritePush(PointerVMSegment, 0)
		default:
			return fmt.Errorf("unexpected keyword %q", token.terminal)
		}
		c.advance()
		return nil
	case IsTerminal(token, "("):
		c.consume("(")
		c.compileExpression()
		c.consume(")")
		return nil
	case isUnaryOp(token):
		op := parseUnaryOp(token)
		c.advance()
		c.compileTerm()
		c.output.WriteArithmetic(op)
		return nil
	default:
		return c.compileVarNameSubterm()
	}

	panic(fmt.Errorf("unexpected token: %q", c.nextToken()))
}

func isBinaryOp(token Token) bool {
	for _, term := range []string{"+", "-", "*", "/", "&", "|", "<", ">", "="} {
		if IsTerminal(token, term) {
			return true
		}
	}
	return false
}

func isUnaryOp(token Token) bool {
	for _, term := range []string{"-", "~"} {
		if IsTerminal(token, term) {
			return true
		}
	}
	return false
}

func parseUnaryOp(token Token) VMOperation {
	switch token.terminal {
	case "-":
		return NegVMOperation
	case "~":
		return NotVMOperation
	}
	return InvalidVMOperation
}

func parseBinaryOp(token Token) VMOperation {
	switch token.terminal {
	case "+":
		return AddVMOperation
	case "-":
		return SubVMOperation
	case "*":
		return MulVMOperation
	case "/":
		return DivVMOperation
	case "&":
		return AndVMOperation
	case "|":
		return OrvMOperation
	case "<":
		return LtVMOperation
	case ">":
		return GtVMOperation
	case "=":
		return EqVMOperation
	}
	return InvalidVMOperation
}

func parseType(token Token) (string, error) {
	if IsTerminal(token, "int", "char", "boolean") {
		return token.terminal, nil
	}
	return parseIdentifier(token)
}

func parseIdentifier(token Token) (string, error) {
	if token.tokenType != Identifier {
		return token.terminal, fmt.Errorf("invalid identifier %q", token.terminal)
	}
	return token.terminal, nil
}

func parseVarName(token Token) (string, error) {
	return parseIdentifier(token)
}

func parseIntegerConstant(token Token) (MachineWord, error) {
	if token.tokenType != IntegerConstant {
		return 0, fmt.Errorf("invalid integer constant %q", token.terminal)
	}
	return token.asInt(), nil
}

func parseStringConstant(token Token) (string, error) {
	if token.tokenType != StringConstant {
		return "", fmt.Errorf("invalid string constant %q", token.terminal)
	}
	return token.terminal, nil
}

func parseSubroutineType(token Token) (s SubroutineType, err error) {
	switch {
	case IsTerminal(token, "function", "constructor", "method"):
		s = SubroutineType(token.terminal)
	default:
		err = fmt.Errorf("Expected \"method\", \"constructor\" or \"function\" but got %s", token.terminal)
	}
	return
}

func formatXML(tag string, content string) string {

	for _, toReplace := range [][]string{{"&", "&amp;"}, {"<", "&lt;"}, {">", "&gt;"}, {"\"", "&quot;"}} {
		content = strings.ReplaceAll(content, toReplace[0], toReplace[1])
	}

	return fmt.Sprintf("<%s> %s </%s>", tag, content, tag)
}
