package lexical

// Lexical analyser implementation, see book @ page 4

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode"
)

// Lexer analyse if a set of tokens is part of our language and
// parse its tokens stream
type Lexer struct {
	program *bytes.Buffer

	identifiers map[string]int

	constants []Constant

	Line           int
	SecondaryToken int
}

// Constant defines a constant type
type Constant struct {
	Type  int
	Value interface{}
}

// NewLexer builds an analyser
func NewLexer(program []byte) *Lexer {
	programBuffer := bytes.NewBuffer(program)
	return &Lexer{
		identifiers: map[string]int{},
		constants:   []Constant{},
		program:     programBuffer,
		Line:        0,
	}
}

// Run runs the lexical analysis
func (a *Lexer) Run() ([]int, error) {
	tokens := []int{}
	for {
		token, err := a.NextToken()
		if err != nil && err != io.EOF {
			return nil, err
		}
		tokens = append(tokens, token)

		if token == EOF {
			break
		}
	}
	return tokens, nil
}

// NextToken returns the next token
func (a *Lexer) NextToken() (int, error) {
	token, err := a.nextToken(a.program)
	if err == io.EOF {
		token = EOF
	}
	return token, nil
}

func (a *Lexer) nextToken(buf *bytes.Buffer) (int, error) {
	var nextRune, nextRune2 rune
	var err error
	token := UNKNOWN

	for {
		nextRune, _, err = buf.ReadRune()
		if err != nil {
			return -1, err
		}

		if nextRune == '\n' {
			a.Line++
		}

		if !unicode.IsSpace(nextRune) {
			break
		}
	}

	if isAlpha(nextRune) {
		text, err := parseWord(buf, func(r rune) bool {
			return isAlphaNumeric(r) || r == '_'
		})

		if err != nil {
			return -1, err
		}

		reservedToken, ok := ReservedWordTokens[text]
		if !ok {
			a.registerIdentifier(text)
			token = ID
		} else {
			token = reservedToken
		}

		buf.UnreadRune()

	} else if isDigit(nextRune) {
		text, err := parseWord(buf, func(r rune) bool {
			return isDigit(r)
		})
		if err != nil {
			return -1, err
		}

		val, _ := strconv.Atoi(text)

		token = Numeral
		a.SecondaryToken = a.addNumeralConstant(val)

		buf.UnreadRune()
	} else if nextRune == '"' {
		buf.ReadRune()
		text, err := parseWord(buf, func(r rune) bool {
			return r != '"'
		})

		if err != nil {
			return -1, err
		}

		token = Stringval
		a.SecondaryToken = a.addStringConstant(text)
	} else {
		switch nextRune {
		case ':':
			token = Colon
			break
		case ';':
			token = Semicolon
			break
		case ',':
			token = Comma
			break
		case '*':
			token = Times
			break
		case '/':
			token = Divide
			break
		case '.':
			token = Dot
			break
		case '[':
			token = LeftSquare
			break
		case ']':
			token = RightSquare
			break
		case '{':
			token = LeftBraces
			break
		case '}':
			token = RightBraces
			break
		case '(':
			token = LeftParenthesis
			break
		case ')':
			token = RightParenthesis
			break
		case '\'':
			runeCtt, _, err := buf.ReadRune()
			if err != nil {
				return -1, err
			}

			expectedQuotes, _, err := buf.ReadRune()
			if err != nil {
				return -1, err
			}

			if expectedQuotes != '\'' {
				return -1, fmt.Errorf("Expected quotes")
			}

			token = Character
			a.SecondaryToken = a.addRuneConstant(runeCtt)
			break
		case '&':
			nextRune2, _, err = buf.ReadRune()
			if err != nil {
				return -1, err
			}
			if nextRune2 != '&' {
				return -1, errors.New("Invalid character")
			}
			token = And
			break
		case '|':
			nextRune2, _, err = buf.ReadRune()
			if err != nil {
				return -1, err
			}
			if nextRune2 != '|' {
				return -1, errors.New("Invalid character")
			}
			token = Or
			break
		case '=':
			nextRune2, _, err = buf.ReadRune()
			if err != nil {
				if err != io.EOF {
					return -1, err
				}
				token = Equals
				break
			}
			if nextRune2 != '=' {
				err = buf.UnreadRune()
				if err != nil {
					return -1, err
				}
				token = Equals
			} else {
				token = EqualEqual
			}
			break
		case '<':
			nextRune2, _, err = buf.ReadRune()
			if err != nil {
				if err != io.EOF {
					return -1, err
				}
				token = LessThan
				break
			}
			if nextRune2 != '=' {
				err = buf.UnreadRune()
				if err != nil {
					return -1, err
				}
				token = LessThan
			} else {
				token = LessOrEqual
			}
		case '>':
			nextRune2, _, err = buf.ReadRune()
			if err != nil {
				if err != io.EOF {
					return -1, err
				}
				token = GreaterThan
				break
			}
			if nextRune2 != '=' {
				err = buf.UnreadRune()
				if err != nil {
					return -1, err
				}
				token = GreaterThan
			} else {
				token = GreaterOrEqual
			}
		case '!':
			nextRune2, _, err = buf.ReadRune()
			if err != nil {
				if err != io.EOF {
					return -1, err
				}

				token = Not
				break
			}
			if nextRune2 != '=' {
				err = buf.UnreadRune()
				if err != nil {
					return -1, err
				}
				token = Not
			} else {
				token = NotEqual
			}
		case '+':
			nextRune2, _, err = buf.ReadRune()
			if err != nil {
				if err != io.EOF {
					return -1, err
				}
				token = Plus
				break
			}
			if nextRune2 != '+' {
				err = buf.UnreadRune()
				if err != nil {
					return -1, err
				}
				token = Plus
			} else {
				token = PlusPlus
			}
		case '-':
			nextRune2, _, err = buf.ReadRune()
			if err != nil {
				if err != io.EOF {
					return -1, err
				}

				token = Minus
				break
			}
			if nextRune2 != '-' {
				err = buf.UnreadRune()
				if err != nil {
					return -1, err
				}
				token = Minus
			} else {
				token = MinusMinus
			}
		}
	}

	return token, nil
}

func parseWord(buf *bytes.Buffer, criteria func(rune) bool) (string, error) {
	var sb strings.Builder
	var err error

	err = buf.UnreadRune()
	if err != nil {
		return "", err
	}

	nextToken, _, err := buf.ReadRune()
	if err != nil {
		return "", err
	}

	for criteria(nextToken) && err != io.EOF {
		sb.WriteRune(nextToken)

		nextToken, _, err = buf.ReadRune()
		if err != nil {
			if err == io.EOF {
				break
			}

			return "", err
		}
	}

	return sb.String(), nil
}

func isAlpha(r rune) bool {
	return unicode.IsLetter(r)
}

func isAlphaNumeric(r rune) bool {
	return isAlpha(r) || isDigit(r)
}

func isDigit(r rune) bool {
	return unicode.IsDigit(r)
}

func (a *Lexer) registerIdentifier(s string) {
	secondaryToken, ok := a.identifiers[s]

	if !ok {
		secondaryToken = len(a.identifiers)
		a.identifiers[s] = secondaryToken
	}

	a.SecondaryToken = secondaryToken
}

// GetRuneConstant returns the rune constant given its id
func (a *Lexer) GetRuneConstant(n int) rune {
	val, _ := a.constants[n].Value.(rune)
	return val
}

// GetStringConstant returns the string constant given its id
func (a *Lexer) GetStringConstant(n int) string {
	val, _ := a.constants[n].Value.(string)
	return val
}

// GetNumeralConstant returns the int constant given its id
func (a *Lexer) GetNumeralConstant(n int) int {
	val, _ := a.constants[n].Value.(int)
	return val
}

// setRuneConstant returns the rune constant given its id
func (a *Lexer) addRuneConstant(n rune) int {
	a.constants = append(a.constants, Constant{
		Type:  Character,
		Value: n,
	})
	return len(a.constants) - 1
}

// addStringConstant returns the string constant given its id
func (a *Lexer) addStringConstant(n string) int {
	a.constants = append(a.constants, Constant{
		Type:  String,
		Value: n,
	})
	return len(a.constants) - 1
}

// addNumeralConstant returns the int constant given its id
func (a *Lexer) addNumeralConstant(n int) int {
	a.constants = append(a.constants, Constant{
		Type:  Numeral,
		Value: n,
	})
	return len(a.constants) - 1
}

// Identifiers retrieves the identifiers
func (a *Lexer) Identifiers() map[string]int {
	return a.identifiers
}
