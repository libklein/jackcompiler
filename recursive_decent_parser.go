package main

import (
	"fmt"
)

type TokenScanner interface {
	Token() Token
	Err() error
	Scan() bool
}

func CompileCode(t TokenScanner) ([]string, error) {
	t.Scan()
	return compileClass(t)
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
		fmt.Println("\tCompiling maybe...")
		code, err := f(t)
		if err != nil {
			return []string{}, nil
		}
		return code, nil
	}
}

func greedy(f CompilationFunction) CompilationFunction {
	return func(t TokenScanner) ([]string, error) {
		fmt.Println("\tCompiling greedily...")
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

func compileTerminal(tokenStream TokenScanner, expectation string) error {
	if token := tokenStream.Token(); token.terminal != expectation {
		return fmt.Errorf("expected terminal %q got %q", expectation, token.terminal)
	}
	if !tokenStream.Scan() && tokenStream.Err() != nil {
		return fmt.Errorf("error receiving next token: %q", tokenStream.Err())
	}
	return nil
}

func newTerminalCompiler(terminal string) CompilationFunction {
	return func(t TokenScanner) (code []string, err error) {
		fmt.Printf("\t\tExpecting terminal %q\n", terminal)
		err = compileTerminal(t, terminal)
		if err == nil {
			fmt.Printf("\tGot terminal %q\n", terminal)
		} else {
			fmt.Printf("\t\t\tError: %v\n", err)
		}
		return
	}
}

func compileClass(tokenStream TokenScanner) (code []string, err error) {
	fmt.Printf("Compiling class\n")
	code, err = chain(
		newTerminalCompiler("class"),
		compileClassName,
		newTerminalCompiler("{"),
		greedy(compileClassVarDec),
		greedy(compileClassSubroutineDec),
		newTerminalCompiler("}"),
	)(tokenStream)

	return
}

func compileClassVarDec(tokenStream TokenScanner) (code []string, err error) {
	fmt.Printf("Compiling class var declaration\n")
	code, err = chain(
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

	return
}

func compileType(tokenStream TokenScanner) ([]string, error) {
	fmt.Printf("Compiling type\n")
	return or(
		newTerminalCompiler("int"),
		newTerminalCompiler("char"),
		newTerminalCompiler("boolean"),
		compileClassName,
	)(tokenStream)
}

func compileClassSubroutineDec(tokenStream TokenScanner) ([]string, error) {
	fmt.Printf("Compiling subroutine dec\n")
	return chain(
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
}

func compileParameterList(tokenStream TokenScanner) ([]string, error) {
	fmt.Printf("Compiling param list\n")
	return maybe(
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
}

func compileSubroutineBody(tokenStream TokenScanner) ([]string, error) {
	fmt.Printf("Compiling subroutine body\n")
	return chain(
		newTerminalCompiler("{"),
		greedy(
			compileVarDec,
		),
		compileStatements,
		newTerminalCompiler("}"),
	)(tokenStream)
}

func compileVarDec(tokenStream TokenScanner) ([]string, error) {
	fmt.Printf("Compiling class var declaration\n")
	return chain(
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
}

func compileIdentifier(tokenStream TokenScanner) ([]string, error) {
	token := tokenStream.Token()
	if token.tokenType != Identifier {
		return nil, fmt.Errorf("invalid identifier %q", token.terminal)
	}
	if !tokenStream.Scan() && tokenStream.Err() != nil {
		return nil, fmt.Errorf("error receiving next token: %q", tokenStream.Err())
	}
	fmt.Printf("\tGot identifier %q\n", token.terminal)
	return []string{token.terminal}, nil
}

func compileClassName(tokenStream TokenScanner) ([]string, error) {
	return compileIdentifier(tokenStream)
}

func compileSubroutineName(tokenStream TokenScanner) ([]string, error) {
	return compileIdentifier(tokenStream)
}

func compileVarName(tokenStream TokenScanner) ([]string, error) {
	return compileIdentifier(tokenStream)
}

func compileStatements(tokenStream TokenScanner) ([]string, error) {
	fmt.Printf("Compiling statements\n")
	return greedy(
		compileStatement,
	)(tokenStream)
}

func compileStatement(tokenStream TokenScanner) ([]string, error) {
	fmt.Printf("Compiling statement\n")
	return or(
		compileLetStatement,
		compileIfStatement,
		compileWhileStatement,
		compileDoStatement,
		compileReturnStatement,
	)(tokenStream)
}

func compileLetStatement(tokenStream TokenScanner) ([]string, error) {
	fmt.Printf("Compiling lets statement\n")
	return chain(
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
}

func compileIfStatement(tokenStream TokenScanner) ([]string, error) {
	fmt.Printf("Compiling if statement\n")
	return chain(
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
}

func compileWhileStatement(tokenStream TokenScanner) ([]string, error) {
	fmt.Printf("Compiling while statement\n")
	return chain(
		newTerminalCompiler("while"),
		newTerminalCompiler("("),
		compileExpression,
		newTerminalCompiler(")"),
		newTerminalCompiler("{"),
		compileStatements,
		newTerminalCompiler("}"),
	)(tokenStream)
}

func compileDoStatement(tokenStream TokenScanner) ([]string, error) {
	fmt.Printf("Compiling do statement\n")
	return chain(
		newTerminalCompiler("do"),
		compileSubroutineCall,
		newTerminalCompiler(";"),
	)(tokenStream)
}

func compileReturnStatement(tokenStream TokenScanner) ([]string, error) {
	fmt.Printf("Compiling return statement\n")
	return chain(
		newTerminalCompiler("return"),
		maybe(compileExpression),
		newTerminalCompiler(";"),
	)(tokenStream)
}

func compileExpression(tokenStream TokenScanner) ([]string, error) {
	fmt.Printf("Compiling expression\n")
	return chain(
		compileTerm,
		greedy(
			chain(
				compileOp,
				compileTerm,
			),
		),
	)(tokenStream)
}

func compileVarNameSubterm(tokenStream TokenScanner) ([]string, error) {
	fmt.Printf("Compiling subterm\n")
	// Parse var name
	varNameCode, err := compileVarName(tokenStream)
	if err != nil {
		return nil, err
	}

	var subtokenCode []string

	fmt.Printf("\t\t\tToken: %v", tokenStream.Token())

	switch tokenStream.Token().terminal {
	case "[":
		subtokenCode, err = chain(
			newTerminalCompiler("["),
			compileExpression,
			newTerminalCompiler("]"),
		)(tokenStream)
	case "(", ".":
		subtokenCode, err = compileCall(tokenStream)
	default:
		subtokenCode = varNameCode
	}
	return append(varNameCode, subtokenCode...), err
}

func compileTerm(tokenStream TokenScanner) ([]string, error) {
	fmt.Printf("Compiling term\n")
	return or(
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
}

func compileCall(tokenStream TokenScanner) ([]string, error) {
	fmt.Printf("Compiling call\n")
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
	fmt.Printf("Compiling subroutine call\n")
	return chain(
		or(
			compileSubroutineName,
			or(compileClassName, compileVarName),
		),
		compileCall,
	)(tokenStream)
}

func compileExpressionList(tokenStream TokenScanner) ([]string, error) {
	fmt.Printf("Compiling expression list\n")
	return maybe(
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
}

func compileOp(tokenStream TokenScanner) ([]string, error) {
	fmt.Printf("Compiling op\n")
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
	fmt.Printf("Compiling unary op\n")
	return or(
		newTerminalCompiler("-"),
		newTerminalCompiler("~"),
	)(tokenStream)
}

func compileKeywordConstant(tokenStream TokenScanner) ([]string, error) {
	fmt.Printf("Compiling kw\n")
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
	fmt.Printf("\tGot integer %d\n", token.asInt())
	return []string{token.terminal}, nil
}

func compileStringConstant(tokenStream TokenScanner) ([]string, error) {
	token := tokenStream.Token()
	if token.tokenType != StringConstant {
		return nil, fmt.Errorf("invalid string constant %q", token.terminal)
	}
	if !tokenStream.Scan() && tokenStream.Err() != nil {
		return nil, fmt.Errorf("error receiving next token: %q", tokenStream.Err())
	}
	fmt.Printf("\tGot integer %s\n", token.terminal)
	return []string{token.terminal}, nil
}
