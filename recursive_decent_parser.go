package main

import (
	"fmt"
	"strings"
)

var (
	symbolTable  SymbolTable = NewSymbolTable()
	lastVarTypes []string
	lastVarKind  SymbolType
	lastVarNames []string
)

func resetCapture() {
	lastVarTypes = []string{}
	lastVarKind = InvalidSymbol
	lastVarNames = []string{}
}

func declareSymbols(scope Scope) {
	if lastVarKind == InvalidSymbol {
		fmt.Println("Tried to parse invalid symbol!")
	}
	if len(lastVarTypes) != len(lastVarNames) && len(lastVarTypes) != 1 {
		fmt.Printf("Tried to parse uneq amount of vars and symbols: %v %v\n", lastVarTypes, lastVarNames)
	}
	for i := 0; i < len(lastVarNames); i++ {
		lastType := lastVarTypes[0]
		if i < len(lastVarTypes) {
			lastType = lastVarTypes[i]
		}
		lastName := lastVarNames[i]
		symbol := symbolTable.Declare(Symbol{symbolType: lastVarKind, variableType: lastType}, lastName, scope)
		fmt.Printf("Declared symbol %q: %v\n", lastName, symbol)
	}
}

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

func CompileCode(t TokenScanner) (code []string, err error) {
	resetCapture()
	t.Scan()
	code, err = compileClass(t)
	return
}

type CompilationFunction func(TokenScanner) ([]string, error)

func chain(funcs ...CompilationFunction) CompilationFunction {
	return func(t TokenScanner) (code []string, err error) {
		var nextCode []string

		for _, f := range funcs {
			nextCode, err = f(t)
			if err != nil {
				return
			}
			code = append(code, nextCode...)
		}

		return
	}
}

func maybe(f CompilationFunction) CompilationFunction {
	return func(t TokenScanner) ([]string, error) {
		code, err := f(t)
		if err != nil {
			return []string{}, nil
		}
		return code, nil
	}
}

func greedy(f CompilationFunction) CompilationFunction {
	return func(t TokenScanner) ([]string, error) {
		code := []string{}
		for {
			nextCode, err := f(t)
			// TODO If is recoverable
			if err != nil {
				return code, nil
			}
			code = append(code, nextCode...)
		}
	}
}

func or(funcs ...CompilationFunction) CompilationFunction {
	return func(t TokenScanner) (code []string, err error) {
		for _, f := range funcs {
			if code, err = f(t); err == nil {
				return
			}
		}
		return
	}
}

func compileTerminal(tokenStream TokenScanner, expectation string) (token Token, err error) {
	token = tokenStream.Token()
	if token.terminal != expectation {
		err = fmt.Errorf("expected terminal %q got %q", expectation, token.terminal)
		return
	}
	if !tokenStream.Scan() && tokenStream.Err() != nil {
		err = fmt.Errorf("error receiving next token: %q", tokenStream.Err())
	}
	return token, err
}

func newTerminalCompiler(terminal string) CompilationFunction {
	return func(t TokenScanner) ([]string, error) {
		token, err := compileTerminal(t, terminal)
		if err == nil {
			return []string{formatXML(string(token.tokenType), token.terminal)}, nil
		}
		return []string{}, err
	}
}

func compileClass(tokenStream TokenScanner) (code []string, err error) {
	symbolTable.Clear(ClassScope)
	childCode, childErr := chain(
		newTerminalCompiler("class"),
		compileClassName,
		newTerminalCompiler("{"),
		greedy(compileClassVarDec),
		greedy(compileClassSubroutineDec),
		newTerminalCompiler("}"),
	)(tokenStream)

	if childErr != nil {
		err = childErr
	} else {
		code = []string{"<class>"}
		code = append(code, childCode...)
		code = append(code, "</class>")
	}

	return
}

func compileClassVarDec(tokenStream TokenScanner) (code []string, err error) {
	fmt.Println("<ClassVarDec>")
	resetCapture()

	lastVarKind = SymbolType(tokenStream.Token().terminal)
	childCode, childErr := chain(
		or(
			newTerminalCompiler("static"),
			newTerminalCompiler("field"),
		),
		compileType,
		compileVarName,
		greedy(
			chain(
				newTerminalCompiler(","),
				compileVarName,
			),
		),
		newTerminalCompiler(";"),
	)(tokenStream)

	// Declare all variables found
	declareSymbols(ClassScope)

	if childErr != nil {
		err = childErr
	} else {
		code = []string{"<classVarDec>"}
		code = append(code, childCode...)
		code = append(code, "</classVarDec>")
	}

	return
}

func compileType(tokenStream TokenScanner) (code []string, err error) {
	lastVarTypes = append(lastVarTypes, tokenStream.Token().terminal)
	if code, err = or(
		newTerminalCompiler("int"),
		newTerminalCompiler("char"),
		newTerminalCompiler("boolean"),
		compileClassName,
	)(tokenStream); err != nil {
		// Remove captured type
		lastVarTypes = lastVarTypes[:len(lastVarTypes)-1]
	}
	return
}

func compileClassSubroutineDec(tokenStream TokenScanner) ([]string, error) {
	childCode, childErr := chain(
		or(
			newTerminalCompiler("constructor"),
			newTerminalCompiler("function"),
			newTerminalCompiler("method"),
		),
		or(
			newTerminalCompiler("void"),
			compileType,
		),
		compileSubroutineName,
		newTerminalCompiler("("),
		compileParameterList,
		newTerminalCompiler(")"),
		compileSubroutineBody,
	)(tokenStream)

	if childErr != nil {
		return []string{}, childErr
	}
	code := []string{"<subroutineDec>"}
	code = append(code, childCode...)
	code = append(code, "</subroutineDec>")
	return code, nil
}

func compileParameterList(tokenStream TokenScanner) ([]string, error) {
	fmt.Println("<ParameterList>")
	resetCapture()

	childCode, childErr := maybe(
		chain(
			compileType,
			compileVarName,
			greedy(
				chain(
					newTerminalCompiler(","),
					compileType,
					compileVarName,
				),
			),
		),
	)(tokenStream)

	// Declare arguments
	if lastVarKind != InvalidSymbol {
		fmt.Println("Uncleared var type", lastVarKind)
	}
	lastVarKind = ArgumentSymbol
	declareSymbols(FunctionScope)

	if childErr != nil {
		return []string{}, childErr
	}
	code := []string{"<parameterList>"}
	code = append(code, childCode...)
	code = append(code, "</parameterList>")
	return code, nil
}

func compileSubroutineBody(tokenStream TokenScanner) ([]string, error) {
	symbolTable.Clear(FunctionScope)
	childCode, childErr := chain(
		newTerminalCompiler("{"),
		greedy(
			compileVarDec,
		),
		compileStatements,
		newTerminalCompiler("}"),
	)(tokenStream)

	if childErr != nil {
		return []string{}, childErr
	}
	code := []string{"<subroutineBody>"}
	code = append(code, childCode...)
	code = append(code, "</subroutineBody>")
	return code, nil
}

func compileVarDec(tokenStream TokenScanner) ([]string, error) {
	fmt.Println("<VarDec>")
	resetCapture()

	childCode, childErr := chain(
		newTerminalCompiler("var"),
		compileType,
		compileVarName,
		greedy(
			chain(
				newTerminalCompiler(","),
				compileVarName,
			),
		),
		newTerminalCompiler(";"),
	)(tokenStream)

	// Declare var
	if lastVarKind != InvalidSymbol {
		fmt.Println("Uncleared var type", lastVarKind)
	}
	lastVarKind = VarSymbol
	declareSymbols(FunctionScope)

	if childErr != nil {
		return []string{}, childErr
	}
	code := []string{"<varDec>"}
	code = append(code, childCode...)
	code = append(code, "</varDec>")
	return code, nil
}

func compileIdentifier(tokenStream TokenScanner) ([]string, error) {
	token := tokenStream.Token()
	if token.tokenType != Identifier {
		return nil, fmt.Errorf("invalid identifier %q", token.terminal)
	}
	if !tokenStream.Scan() && tokenStream.Err() != nil {
		return nil, fmt.Errorf("error receiving next token: %q", tokenStream.Err())
	}
	return []string{formatXML("identifier", token.terminal)}, nil
}

func compileClassName(tokenStream TokenScanner) ([]string, error) {
	return compileIdentifier(tokenStream)
}

func compileSubroutineName(tokenStream TokenScanner) ([]string, error) {
	return compileIdentifier(tokenStream)
}

func compileVarName(tokenStream TokenScanner) (code []string, err error) {
	lastVarNames = append(lastVarNames, tokenStream.Token().terminal)
	if code, err = compileIdentifier(tokenStream); err != nil {
		lastVarNames = lastVarNames[:len(lastVarNames)-1]
	}
	return
}

func compileStatements(tokenStream TokenScanner) ([]string, error) {
	childCode, childErr := greedy(
		compileStatement,
	)(tokenStream)

	if childErr != nil {
		return []string{}, childErr
	}
	code := []string{"<statements>"}
	code = append(code, childCode...)
	code = append(code, "</statements>")
	return code, nil
}

func compileStatement(tokenStream TokenScanner) ([]string, error) {
	childCode, childErr := or(
		compileLetStatement,
		compileIfStatement,
		compileWhileStatement,
		compileDoStatement,
		compileReturnStatement,
	)(tokenStream)

	if childErr != nil {
		return []string{}, childErr
	}

	return childCode, nil
}

func compileLetStatement(tokenStream TokenScanner) ([]string, error) {
	childCode, childErr := chain(
		newTerminalCompiler("let"),
		compileVarName,
		maybe(
			chain(
				newTerminalCompiler("["),
				compileExpression,
				newTerminalCompiler("]"),
			),
		),
		newTerminalCompiler("="),
		compileExpression,
		newTerminalCompiler(";"),
	)(tokenStream)

	if childErr != nil {
		return []string{}, childErr
	}
	code := []string{"<letStatement>"}
	code = append(code, childCode...)
	code = append(code, "</letStatement>")
	return code, nil
}

func compileIfStatement(tokenStream TokenScanner) ([]string, error) {
	childCode, childErr := chain(
		newTerminalCompiler("if"),
		newTerminalCompiler("("),
		compileExpression,
		newTerminalCompiler(")"),
		newTerminalCompiler("{"),
		compileStatements,
		newTerminalCompiler("}"),
		maybe(
			chain(
				newTerminalCompiler("else"),
				newTerminalCompiler("{"),
				compileStatements,
				newTerminalCompiler("}"),
			),
		),
	)(tokenStream)

	if childErr != nil {
		return []string{}, childErr
	}
	code := []string{"<ifStatement>"}
	code = append(code, childCode...)
	code = append(code, "</ifStatement>")
	return code, nil
}

func compileWhileStatement(tokenStream TokenScanner) ([]string, error) {
	childCode, childErr := chain(
		newTerminalCompiler("while"),
		newTerminalCompiler("("),
		compileExpression,
		newTerminalCompiler(")"),
		newTerminalCompiler("{"),
		compileStatements,
		newTerminalCompiler("}"),
	)(tokenStream)

	if childErr != nil {
		return []string{}, childErr
	}
	code := []string{"<whileStatement>"}
	code = append(code, childCode...)
	code = append(code, "</whileStatement>")
	return code, nil
}

func compileDoStatement(tokenStream TokenScanner) ([]string, error) {
	childCode, childErr := chain(
		newTerminalCompiler("do"),
		compileSubroutineCall,
		newTerminalCompiler(";"),
	)(tokenStream)

	if childErr != nil {
		return []string{}, childErr
	}
	code := []string{"<doStatement>"}
	code = append(code, childCode...)
	code = append(code, "</doStatement>")
	return code, nil
}

func compileReturnStatement(tokenStream TokenScanner) ([]string, error) {
	childCode, childErr := chain(
		newTerminalCompiler("return"),
		maybe(compileExpression),
		newTerminalCompiler(";"),
	)(tokenStream)

	if childErr != nil {
		return []string{}, childErr
	}
	code := []string{"<returnStatement>"}
	code = append(code, childCode...)
	code = append(code, "</returnStatement>")
	return code, nil
}

func compileExpression(tokenStream TokenScanner) ([]string, error) {
	childCode, childErr := chain(
		compileTerm,
		greedy(
			chain(
				compileOp,
				compileTerm,
			),
		),
	)(tokenStream)

	if childErr != nil {
		return []string{}, childErr
	}
	code := []string{"<expression>"}
	code = append(code, childCode...)
	code = append(code, "</expression>")
	return code, nil
}

func compileVarNameSubterm(tokenStream TokenScanner) ([]string, error) {
	// Parse var name
	varNameCode, err := compileVarName(tokenStream)
	if err != nil {
		return nil, err
	}

	var subtokenCode []string

	switch tokenStream.Token().terminal {
	case "[":
		subtokenCode, err = chain(
			newTerminalCompiler("["),
			compileExpression,
			newTerminalCompiler("]"),
		)(tokenStream)
	case "(", ".":
		subtokenCode, err = compileCall(tokenStream)
	}
	return append(varNameCode, subtokenCode...), err
}

func compileTerm(tokenStream TokenScanner) ([]string, error) {
	childCode, childErr := or(
		compileIntegerConstant,
		compileStringConstant,
		compileKeywordConstant,
		chain(
			newTerminalCompiler("("),
			compileExpression,
			newTerminalCompiler(")"),
		),
		chain(
			compileUnaryOp,
			compileTerm,
		),
		// Try last, this will consume a token
		compileVarNameSubterm,
	)(tokenStream)

	if childErr != nil {
		return []string{}, childErr
	}
	code := []string{"<term>"}
	code = append(code, childCode...)
	code = append(code, "</term>")
	return code, nil
}

func compileCall(tokenStream TokenScanner) ([]string, error) {
	return or(
		chain(
			newTerminalCompiler("("),
			compileExpressionList,
			newTerminalCompiler(")"),
		),
		chain(
			newTerminalCompiler("."),
			compileSubroutineName,
			newTerminalCompiler("("),
			compileExpressionList,
			newTerminalCompiler(")"),
		),
	)(tokenStream)
}

func compileSubroutineCall(tokenStream TokenScanner) ([]string, error) {
	return chain(
		or(
			compileSubroutineName,
			or(compileClassName, compileVarName),
		),
		compileCall,
	)(tokenStream)
}

func compileExpressionList(tokenStream TokenScanner) ([]string, error) {
	childCode, childErr := maybe(
		chain(
			compileExpression,
			greedy(
				chain(
					newTerminalCompiler(","),
					compileExpression,
				),
			),
		),
	)(tokenStream)

	if childErr != nil {
		return []string{}, childErr
	}
	code := []string{"<expressionList>"}
	code = append(code, childCode...)
	code = append(code, "</expressionList>")
	return code, nil
}

func compileOp(tokenStream TokenScanner) ([]string, error) {
	return or(
		newTerminalCompiler("+"),
		newTerminalCompiler("-"),
		newTerminalCompiler("*"),
		newTerminalCompiler("/"),
		newTerminalCompiler("&"),
		newTerminalCompiler("|"),
		newTerminalCompiler("<"),
		newTerminalCompiler(">"),
		newTerminalCompiler("="),
	)(tokenStream)
}

func compileUnaryOp(tokenStream TokenScanner) ([]string, error) {
	return or(
		newTerminalCompiler("-"),
		newTerminalCompiler("~"),
	)(tokenStream)
}

func compileKeywordConstant(tokenStream TokenScanner) ([]string, error) {
	return or(
		newTerminalCompiler("true"),
		newTerminalCompiler("false"),
		newTerminalCompiler("null"),
		newTerminalCompiler("this"),
	)(tokenStream)
}

func compileIntegerConstant(tokenStream TokenScanner) ([]string, error) {
	token := tokenStream.Token()
	if token.tokenType != IntegerConstant {
		return nil, fmt.Errorf("invalid integer constant %q", token.terminal)
	}
	if !tokenStream.Scan() && tokenStream.Err() != nil {
		return nil, fmt.Errorf("error receiving next token: %q", tokenStream.Err())
	}
	return []string{formatXML("integerConstant", token.terminal)}, nil
}

func compileStringConstant(tokenStream TokenScanner) ([]string, error) {
	token := tokenStream.Token()
	if token.tokenType != StringConstant {
		return nil, fmt.Errorf("invalid string constant %q", token.terminal)
	}
	if !tokenStream.Scan() && tokenStream.Err() != nil {
		return nil, fmt.Errorf("error receiving next token: %q", tokenStream.Err())
	}
	return []string{formatXML("stringConstant", token.terminal)}, nil
}

func formatXML(tag string, content string) string {

	for _, toReplace := range [][]string{{"&", "&amp;"}, {"<", "&lt;"}, {">", "&gt;"}, {"\"", "&quot;"}} {
		content = strings.ReplaceAll(content, toReplace[0], toReplace[1])
	}

	return fmt.Sprintf("<%s> %s </%s>", tag, content, tag)
}
