package syntax

import (
	"fmt"

	"github.com/shanepadgett/pxlc/internal/diagnostic"
)

// Parse builds an unvalidated declaration tree from source.
func Parse(path string, source []byte, maximumTokens int) (*Document, []diagnostic.Diagnostic) {
	tokens, diagnostics := lex(path, source, maximumTokens)
	if len(diagnostics) != 0 {
		return nil, diagnostics
	}
	p := parser{tokens: tokens}
	doc, diag := p.document()
	if diag != nil {
		return nil, []diagnostic.Diagnostic{*diag}
	}
	return doc, nil
}

type parser struct {
	tokens []token
	index  int
}

func (p *parser) document() (*Document, *diagnostic.Diagnostic) {
	if diag := p.expectKeyword("pxlc"); diag != nil {
		return nil, diag
	}
	version, diag := p.expectValue(tokenWord, "source-format version")
	if diag != nil {
		return nil, diag
	}
	doc := &Document{Version: version}
	for p.peek().kind != tokenEOF {
		keyword := p.next()
		if keyword.kind != tokenWord {
			return nil, p.unexpected(keyword, "declaration")
		}
		var declarationDiagnostic *diagnostic.Diagnostic
		switch keyword.text {
		case "asset":
			var value Value
			value, declarationDiagnostic = p.expectValue(tokenWord, "asset name")
			if declarationDiagnostic == nil {
				doc.Assets = append(doc.Assets, value)
			}
		case "canvas":
			var value Canvas
			value, declarationDiagnostic = p.canvas(keyword)
			if declarationDiagnostic == nil {
				doc.Canvases = append(doc.Canvases, value)
			}
		case "palette":
			var value Palette
			value, declarationDiagnostic = p.palette(keyword)
			if declarationDiagnostic == nil {
				doc.Palettes = append(doc.Palettes, value)
			}
		case "background":
			var value Background
			value, declarationDiagnostic = p.background(keyword)
			if declarationDiagnostic == nil {
				doc.Backgrounds = append(doc.Backgrounds, value)
			}
		case "layer":
			var value Layer
			value, declarationDiagnostic = p.layer(keyword)
			if declarationDiagnostic == nil {
				doc.Layers = append(doc.Layers, value)
			}
		default:
			declarationDiagnostic = p.unexpected(keyword, "asset, canvas, palette, background, or layer")
		}
		if declarationDiagnostic != nil {
			return nil, declarationDiagnostic
		}
	}
	return doc, nil
}

func (p *parser) canvas(start token) (Canvas, *diagnostic.Diagnostic) {
	width, diag := p.expectValue(tokenWord, "canvas width")
	if diag != nil {
		return Canvas{}, diag
	}
	height, diag := p.expectValue(tokenWord, "canvas height")
	if diag != nil {
		return Canvas{}, diag
	}
	return Canvas{Width: width, Height: height, Span: joinedSpan(start.span, height.Span)}, nil
}

func (p *parser) palette(start token) (Palette, *diagnostic.Diagnostic) {
	name, diag := p.expectValue(tokenWord, "palette name")
	if diag != nil {
		return Palette{}, diag
	}
	palette := Palette{Name: name}
	if p.peek().kind == tokenWord && p.peek().text == "max" {
		p.next()
		maximum, maximumDiagnostic := p.expectValue(tokenWord, "palette maximum")
		if maximumDiagnostic != nil {
			return Palette{}, maximumDiagnostic
		}
		palette.Maximum = &maximum
	}
	if diag = p.expectKind(tokenLeftBrace, "{"); diag != nil {
		return Palette{}, diag
	}
	for p.peek().kind != tokenRightBrace {
		if p.peek().kind == tokenEOF {
			return Palette{}, p.unexpected(p.peek(), "palette entry or }")
		}
		entry, entryDiagnostic := p.paletteEntry()
		if entryDiagnostic != nil {
			return Palette{}, entryDiagnostic
		}
		palette.Entries = append(palette.Entries, entry)
	}
	end := p.next()
	palette.Span = joinedSpan(start.span, end.span)
	return palette, nil
}

func (p *parser) paletteEntry() (PaletteEntry, *diagnostic.Diagnostic) {
	start := p.next()
	if start.kind != tokenWord || (start.text != "color" && start.text != "transparent") {
		return PaletteEntry{}, p.unexpected(start, "color or transparent")
	}
	name, diag := p.expectValue(tokenWord, "color name")
	if diag != nil {
		return PaletteEntry{}, diag
	}
	symbol, diag := p.expectValue(tokenString, "one-character grid symbol")
	if diag != nil {
		return PaletteEntry{}, diag
	}
	entry := PaletteEntry{Name: name, Symbol: symbol, Transparent: start.text == "transparent"}
	end := symbol.Span
	if !entry.Transparent {
		hex, hexDiagnostic := p.expectValue(tokenWord, "#RRGGBB color")
		if hexDiagnostic != nil {
			return PaletteEntry{}, hexDiagnostic
		}
		entry.Hex = &hex
		end = hex.Span
	}
	entry.Span = joinedSpan(start.span, end)
	return entry, nil
}

func (p *parser) background(start token) (Background, *diagnostic.Diagnostic) {
	palette, diag := p.expectValue(tokenWord, "background palette")
	if diag != nil {
		return Background{}, diag
	}
	color, diag := p.expectValue(tokenWord, "background color")
	if diag != nil {
		return Background{}, diag
	}
	return Background{Palette: palette, Color: color, Span: joinedSpan(start.span, color.Span)}, nil
}

func (p *parser) layer(start token) (Layer, *diagnostic.Diagnostic) {
	name, diag := p.expectValue(tokenWord, "layer name")
	if diag != nil {
		return Layer{}, diag
	}
	if diag = p.expectKeyword("using"); diag != nil {
		return Layer{}, diag
	}
	palette, diag := p.expectValue(tokenWord, "layer palette")
	if diag != nil {
		return Layer{}, diag
	}
	if diag = p.expectKind(tokenLeftBrace, "{"); diag != nil {
		return Layer{}, diag
	}
	layer := Layer{Name: name, Palette: palette}
	for p.peek().kind != tokenRightBrace {
		if p.peek().kind == tokenEOF {
			return Layer{}, p.unexpected(p.peek(), "drawing operation or }")
		}
		op, operationDiagnostic := p.operation()
		if operationDiagnostic != nil {
			return Layer{}, operationDiagnostic
		}
		layer.Operations = append(layer.Operations, op)
	}
	end := p.next()
	layer.Span = joinedSpan(start.span, end.span)
	return layer, nil
}

func (p *parser) operation() (Operation, *diagnostic.Diagnostic) {
	start := p.next()
	if start.kind != tokenWord {
		return Operation{}, p.unexpected(start, "drawing operation")
	}
	var kind OperationKind
	switch start.text {
	case "pixel":
		kind = OperationPixel
	case "hspan":
		kind = OperationHSpan
	case "vspan":
		kind = OperationVSpan
	case "rect":
		kind = OperationRect
	case "grid":
		kind = OperationGrid
	default:
		return Operation{}, p.unexpected(start, "pixel, hspan, vspan, rect, or grid")
	}
	x, diag := p.expectValue(tokenWord, "x coordinate")
	if diag != nil {
		return Operation{}, diag
	}
	y, diag := p.expectValue(tokenWord, "y coordinate")
	if diag != nil {
		return Operation{}, diag
	}
	op := Operation{Kind: kind, X: x, Y: y}
	if kind == OperationGrid {
		if diag = p.expectKind(tokenLeftBrace, "{"); diag != nil {
			return Operation{}, diag
		}
		for p.peek().kind != tokenRightBrace {
			row, rowDiagnostic := p.expectValue(tokenString, "grid row")
			if rowDiagnostic != nil {
				return Operation{}, rowDiagnostic
			}
			op.Rows = append(op.Rows, row)
		}
		end := p.next()
		op.Span = joinedSpan(start.span, end.span)
		return op, nil
	}
	if kind == OperationHSpan || kind == OperationVSpan || kind == OperationRect {
		op.Width, diag = p.expectValue(tokenWord, "length or width")
		if diag != nil {
			return Operation{}, diag
		}
	}
	if kind == OperationRect {
		op.Height, diag = p.expectValue(tokenWord, "rectangle height")
		if diag != nil {
			return Operation{}, diag
		}
	}
	op.Color, diag = p.expectValue(tokenWord, "color name")
	if diag != nil {
		return Operation{}, diag
	}
	op.Span = joinedSpan(start.span, op.Color.Span)
	return op, nil
}

func (p *parser) expectKeyword(want string) *diagnostic.Diagnostic {
	tok := p.next()
	if tok.kind != tokenWord || tok.text != want {
		return p.unexpected(tok, want)
	}
	return nil
}

func (p *parser) expectKind(kind tokenKind, want string) *diagnostic.Diagnostic {
	tok := p.next()
	if tok.kind != kind {
		return p.unexpected(tok, want)
	}
	return nil
}

func (p *parser) expectValue(kind tokenKind, want string) (Value, *diagnostic.Diagnostic) {
	tok := p.next()
	if tok.kind != kind {
		return Value{}, p.unexpected(tok, want)
	}
	return Value{Text: tok.text, Span: tok.span}, nil
}

func (p *parser) unexpected(got token, want string) *diagnostic.Diagnostic {
	found := got.text
	if got.kind == tokenEOF {
		found = "end of file"
	}
	d := diagnostic.Error(got.span, "PXLC-E001", fmt.Sprintf("expected %s; found %q", want, found))
	return &d
}

func (p *parser) peek() token {
	return p.tokens[p.index]
}

func (p *parser) next() token {
	tok := p.tokens[p.index]
	if tok.kind != tokenEOF {
		p.index++
	}
	return tok
}

func joinedSpan(start, end diagnostic.Span) diagnostic.Span {
	return diagnostic.Span{Path: start.Path, Start: start.Start, End: end.End}
}
