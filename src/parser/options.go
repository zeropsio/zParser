package parser

type OptionFunc func(p *Parser)

type MultiLineOutputHandling int

const (
	// MultilineToOneLine squashes output into a single line with `\n` characters instead of newline
	MultilineToOneLine MultiLineOutputHandling = iota
	// MultilinePreserved preservers output with no modifications
	MultilinePreserved
	// MultilineWithIndent preserves multiline output, but adds indentation, so all lines are the same (used for yaml)
	MultilineWithIndent
)

func WithMultilineOutputHandling(handling MultiLineOutputHandling) OptionFunc {
	return func(p *Parser) {
		p.multiLineOutputHandling = handling
	}
}

func WithMaxFunctionCount(c int) OptionFunc {
	return func(p *Parser) {
		p.maxFunctionCount = c
	}
}
