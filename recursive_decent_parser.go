package main

import (
	"fmt"
	"strconv"
	"strings"
)

var (
	symbolTable  SymbolTable = NewSymbolTable()
	lastVarTypes []string
	lastVarKind  SymbolType
	lastVarNames []string
)

/*
* TODO
* Clear table on subroutine entry: DONE
* Declare variables: DONE
* Lookup variables
*
* Problem: When im parsing a variable declaration, i have no way of knowing what the identifier, type, etc is
*
 */

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
	// TODO Emit code
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
	fmt.Printf("Done with compilation of class!\n")
	// TODO EOF
	c.consume("}")
}

func (c *JackCompiler) compileClassVarDec() error {
	switch token := c.nextToken(); {
	case IsTerminal(token, "static"):
		c.consume("static")
		c.compileVarSequence(StaticSymbol)
	case IsTerminal(token, "field"):
		c.consume("field")
		c.compileVarSequence(FieldSymbol)
	default:
		return fmt.Errorf("Expected \"static\" or \"field\" but got %s", token.terminal)
	}
	return nil
}

func (c *JackCompiler) compileVarSequence(symbolType SymbolType) (numDeclarations MachineWord) {
	symbol := Symbol{symbolType: symbolType}

	symbol.variableType, _ = parseType(c.nextToken())
	c.consume()

	for {
		varName, _ := parseIdentifier(c.nextToken())
		c.consume() // consume identifier

		numDeclarations += 1

		// Register types in symbol table
		c.symbolTable.Declare(symbol, varName, ClassScope)
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

	switch token := c.nextToken(); {
	case IsTerminal(token, "function"):

	case IsTerminal(token, "constructor"):
	case IsTerminal(token, "method"):
	default:
		return fmt.Errorf("Expected \"method\", \"constructor\" or \"function\" but got %s", c.nextToken().terminal)
	}

	//returnType, _ := parseType(c.advance())
	c.consume()
	name, _ := parseIdentifier(c.advance())
	c.consume() // Consume identfier

	c.consume("(")

	if !IsTerminal(c.nextToken(), ")") {
		c.compileParameterList()
	}

	c.consume(")")

	c.compileSubroutine(name)

	return nil
}

func (c *JackCompiler) compileSubroutine(name string) {
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
	return c.compileVarSequence(VarSymbol)
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

func (c *JackCompiler) compileArrayAccess(name string) {
	c.consume("[")
	// Stores offset on top of stack
	c.compileExpression()
	c.consume("]")

	// Emit code that moves the that pointer
	// Store base addr on stack
	segment, index := c.generateVariableAccess(name)
	c.output.WritePush(segment, index)
	// Add together
	c.output.WriteArithmetic(AddVMOperation)
	// Pop into pointer (THAT)
	c.output.WritePop(PointerVMSegment, 1)
	// Pop value onto stack
	c.output.WritePop(ThatVMSegment, 0)
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

	if IsTerminal(c.advance(), "[") {
		// TODO Array access
		panic("Array let not implemented")
		c.consume("[")
		c.compileExpression()
		c.consume("]")
		c.compileArrayAccess(varName)
	}
	c.consume("=")
	c.compileExpression()
	c.consume(";")
	// Layout: Value of expression is on top of stack.
	//		   -> Pop last value into var Name
	// TODO Array access
	segment, index := c.generateVariableAccess(varName)
	c.output.WritePop(segment, index)
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
		// TODO Check if name is a symbol! If it is, move this pointer accordingly
		c.consume(".")
		methodName, err := parseIdentifier(c.nextToken())
		if err != nil {
			panic(err)
		}
		name = name + "." + methodName
		// Advance over identifier
		c.advance()
		fallthrough
	case "(":
		c.consume("(")
		nargs := c.compileExpressionList()
		c.output.WriteCall(name, nargs)
		c.consume(")")
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
		c.compileArrayAccess(varName)
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
			// TODO Emit Code
			panic("Not implemented!")
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

func formatXML(tag string, content string) string {

	for _, toReplace := range [][]string{{"&", "&amp;"}, {"<", "&lt;"}, {">", "&gt;"}, {"\"", "&quot;"}} {
		content = strings.ReplaceAll(content, toReplace[0], toReplace[1])
	}

	return fmt.Sprintf("<%s> %s </%s>", tag, content, tag)
}
