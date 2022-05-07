package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

var (
	keywordRegex         = regexp.MustCompile(`(class|constructor|function|method|field|static|var|int|char|boolean|void|true|false|null|this|let|do|if|else|while|return)`)
	symbolRegex          = regexp.MustCompile(`[\{\}\[\]\(\)\.\,\;\+\-\*\/\&\|\<\>\)\=\~]`)
	integerConstantRegex = regexp.MustCompile(`\d{1,5}`)
	stringConstantRegex  = regexp.MustCompile(`"[^"\n]*"`)
	identifierRegex      = regexp.MustCompile(`[a-zA-Z_]\w*`)
	regexes              = []*regexp.Regexp{keywordRegex, symbolRegex, integerConstantRegex, stringConstantRegex, identifierRegex}
	whitespaceRegex      = regexp.MustCompile(`^\s*$`)

	regexTokenTypeMapping = map[*regexp.Regexp]TokenType{
		keywordRegex:         Keyword,
		symbolRegex:          SymbolTokenType,
		integerConstantRegex: IntegerConstant,
		stringConstantRegex:  StringConstant,
		identifierRegex:      Identifier,
	}
)

type FilteredReader struct {
	reader *bufio.Reader
}

func NewFilteredReader(r io.Reader) FilteredReader {
	return FilteredReader{reader: bufio.NewReader(r)}
}

func (r *FilteredReader) Read(b []byte) (int, error) {
	var (
		err  error
		char rune
		n    int
	)

	i := 0
	for i < cap(b) {
		char, n, err = r.reader.ReadRune()

		if n == 0 {
			break
		}

		if err != nil && !errors.Is(err, io.EOF) {
			return 0, err
		}

		if char == '/' {
			nextChar, _, nextErr := r.reader.ReadRune()
			if nextErr != nil {
				if !errors.Is(nextErr, io.EOF) {
					return i, nextErr
				} else {
					err = io.EOF
				}
			} else if nextChar == '/' {
				// Discard until newline character
				_, err := r.reader.ReadString('\n')
				if err != nil {
					return i, err
				}
				continue
			} else if nextChar == '*' {
				// Discard until */
				for {
					str, err := r.reader.ReadString('/')
					if err != nil {
						return i, fmt.Errorf("Unclosed comment! (%v)", err)
					}
					if len(str) == 0 {
						return i, fmt.Errorf("Unclosed comment!")
					}
					if str[len(str)-2] == '*' {
						break
					}
				}
				continue
			} else {
				unreadErr := r.reader.UnreadRune()
				if unreadErr != nil {
					return i, unreadErr
				} else {
					// Reset EOF error
					err = nil
				}
			}
		}

		if n == 0 {
			return n, err
		} else if i+n <= len(b) {
			i += utf8.EncodeRune(b[i:], char)
			if errors.Is(err, io.EOF) {
				break
			}
		} else {
			unreadErr := r.reader.UnreadRune()
			if unreadErr != nil {
				return i, nil
			}
		}

	}

	return i, err
}

type Tokenizer struct {
	scanner   *bufio.Scanner
	nextToken Token
	err       error
}

func NewTokenizer(r io.Reader) Tokenizer {
	// Switch regex to longest mode
	for _, regex := range regexes {
		regex.Longest()
	}

	commentFilter := NewFilteredReader(r)
	scanner := bufio.NewScanner(&commentFilter)
	scanner.Split(splitToken)
	return Tokenizer{scanner: scanner}
}

func matchToken(line string) ([]int, error) {
	minRegexIndex := len(regexes)
	minRegexMatch := []int{10000, 10000}
	for i, regex := range regexes {
		if match := regex.FindStringIndex(line); match != nil && match[0] <= minRegexMatch[0] {
			if match[0] < minRegexMatch[0] || (match[1]-match[0]) > (minRegexMatch[1]-minRegexMatch[0]) {
				minRegexIndex = i
				minRegexMatch = match
			}
		}
	}

	if minRegexIndex == len(regexes) {
		return []int{}, fmt.Errorf("Unknown token %q", line)
	}

	// Check if only whitespace
	if !whitespaceRegex.MatchString(line[0:minRegexMatch[0]]) {
		return []int{}, fmt.Errorf("Could not parse %q, matched %q but %q contains characters", line, line[minRegexMatch[0]:minRegexMatch[1]], line[0:minRegexMatch[0]])
	}

	return append(minRegexMatch, minRegexIndex), nil
}

func splitToken(data []byte, atEOF bool) (advance int, token []byte, err error) {
	dataString := strings.TrimLeftFunc(string(data), unicode.IsSpace)
	if len(dataString) == 0 {
		advance = 0
		token = nil
		return
	}

	matchIndex, matchErr := matchToken(dataString)

	if matchErr != nil {
		if atEOF {
			err = fmt.Errorf("Unkown token %q", token)
		} else {
			advance = 0
			token = nil
		}
		return
	}

	matchBegin := matchIndex[0]
	matchEnd := matchIndex[1]

	advance = matchEnd + (len(data) - len(dataString))
	token = []byte(dataString[matchBegin:matchEnd])

	return
}

func parseToken(tokenString string) (token Token, err error) {
	var regexMatch []int
	regexMatch, err = matchToken(tokenString)
	if err != nil {
		return
	}

	regexIndex := regexMatch[2]

	token.terminal = tokenString

	switch regexIndex {
	case 0:
		token.tokenType = Keyword
	case 1:
		token.tokenType = SymbolTokenType
	case 2:
		token.tokenType = IntegerConstant
	case 3:
		token.tokenType = StringConstant
		token.terminal = tokenString[1 : len(tokenString)-1]
	case 4:
		token.tokenType = Identifier
	default:
		err = fmt.Errorf("Unknown token %q", tokenString)
	}

	return
}

func (t *Tokenizer) Err() error {
	return t.err
}

func (t *Tokenizer) Scan() bool {
	for t.scanner.Scan() {
		tokenString := t.scanner.Text()
		// Parse instruction
		token, err := parseToken(tokenString)
		if err != nil {
			t.err = err
			return false
		}
		t.nextToken = token
		return true
	}

	return false
}

func (t *Tokenizer) Token() Token {
	return t.nextToken
}
