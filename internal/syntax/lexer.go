package syntax

import (
	"fmt"

	"github.com/shanepadgett/pxlc/internal/diagnostic"
)

type tokenKind uint8

const (
	tokenEOF tokenKind = iota
	tokenWord
	tokenString
	tokenLeftBrace
	tokenRightBrace
)

type token struct {
	kind tokenKind
	text string
	span diagnostic.Span
}

type lexer struct {
	path   string
	source []byte
	index  int
	line   int
	column int
}

func lex(path string, source []byte, maximumTokens int) ([]token, []diagnostic.Diagnostic) {
	l := lexer{path: path, source: source, line: 1, column: 1}
	tokens := make([]token, 0, min(len(source)/3, maximumTokens))
	for {
		tok, diag := l.next()
		if diag != nil {
			return nil, []diagnostic.Diagnostic{*diag}
		}
		if tok.kind != tokenEOF && len(tokens) >= maximumTokens {
			d := diagnostic.Error(tok.span, "PXLC-E024", fmt.Sprintf("source exceeds the limit of %d tokens", maximumTokens))
			return nil, []diagnostic.Diagnostic{d}
		}
		tokens = append(tokens, tok)
		if tok.kind == tokenEOF {
			return tokens, nil
		}
	}
}

func (l *lexer) next() (token, *diagnostic.Diagnostic) {
	if diag := l.skipSpaceAndComments(); diag != nil {
		return token{}, diag
	}
	start := l.position()
	if l.index >= len(l.source) {
		return token{kind: tokenEOF, span: l.span(start)}, nil
	}

	switch l.source[l.index] {
	case '{':
		l.advance()
		return token{kind: tokenLeftBrace, text: "{", span: l.span(start)}, nil
	case '}':
		l.advance()
		return token{kind: tokenRightBrace, text: "}", span: l.span(start)}, nil
	case '"':
		return l.stringToken(start)
	default:
		return l.wordToken(start)
	}
}

func (l *lexer) skipSpaceAndComments() *diagnostic.Diagnostic {
	for {
		for l.index < len(l.source) && isSpace(l.source[l.index]) {
			l.advance()
		}
		if l.index+1 >= len(l.source) || l.source[l.index] != '/' || l.source[l.index+1] != '/' {
			return nil
		}
		for l.index < len(l.source) && l.source[l.index] != '\n' {
			b := l.source[l.index]
			if b != '\t' && b != '\r' && (b < 0x20 || b > 0x7e) {
				start := l.position()
				l.advance()
				d := diagnostic.Error(l.span(start), "PXLC-E001", "comments must contain printable ASCII")
				return &d
			}
			l.advance()
		}
	}
}

func (l *lexer) stringToken(start diagnostic.Position) (token, *diagnostic.Diagnostic) {
	l.advance()
	contentStart := l.index
	for l.index < len(l.source) && l.source[l.index] != '"' {
		b := l.source[l.index]
		if b < 0x20 || b > 0x7e || b == '\\' {
			span := l.span(start)
			d := diagnostic.Error(span, "PXLC-E001", "strings must contain unescaped printable ASCII")
			return token{}, &d
		}
		l.advance()
	}
	if l.index >= len(l.source) {
		span := l.span(start)
		d := diagnostic.Error(span, "PXLC-E001", "unterminated string")
		return token{}, &d
	}
	text := string(l.source[contentStart:l.index])
	l.advance()
	return token{kind: tokenString, text: text, span: l.span(start)}, nil
}

func (l *lexer) wordToken(start diagnostic.Position) (token, *diagnostic.Diagnostic) {
	contentStart := l.index
	for l.index < len(l.source) {
		b := l.source[l.index]
		if isSpace(b) || b == '{' || b == '}' || b == '"' {
			break
		}
		if b == '/' && l.index+1 < len(l.source) && l.source[l.index+1] == '/' {
			break
		}
		if b < 0x21 || b > 0x7e {
			span := l.span(start)
			d := diagnostic.Error(span, "PXLC-E001", "source tokens must use printable ASCII")
			return token{}, &d
		}
		l.advance()
	}
	if contentStart == l.index {
		l.advance()
		span := l.span(start)
		d := diagnostic.Error(span, "PXLC-E001", "invalid source character")
		return token{}, &d
	}
	return token{kind: tokenWord, text: string(l.source[contentStart:l.index]), span: l.span(start)}, nil
}

func (l *lexer) advance() {
	if l.source[l.index] == '\n' {
		l.line++
		l.column = 1
	} else {
		l.column++
	}
	l.index++
}

func (l *lexer) position() diagnostic.Position {
	return diagnostic.Position{Offset: l.index, Line: l.line, Column: l.column}
}

func (l *lexer) span(start diagnostic.Position) diagnostic.Span {
	return diagnostic.Span{Path: l.path, Start: start, End: l.position()}
}

func isSpace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\r' || b == '\n'
}
