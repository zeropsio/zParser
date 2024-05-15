package parser

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/zeropsio/zParser/v2/src/functions"
	"github.com/zeropsio/zParser/v2/src/metaError"
	"github.com/zeropsio/zParser/v2/src/modifiers"
)

const (
	escapeChar     = '\\'
	newlineChar    = '\n'
	itemStartChar  = '<'
	itemEndChar    = '>'
	funcStartChar  = '@' // combined with itemStartChar which must be preceding funcStartChar
	modifierChar   = '|'
	paramStartChar = '('
	paramEndChar   = ')'
	paramSepChar   = ','
)

type Parser struct {
	out *bufio.Writer
	in  *bufio.Reader

	// Limits maximum amount of function calls (by default set to -1 which means unlimited)
	maxFunctionCount int
	// Sets how multiline output of functions will be formatted, defaults to no modifications
	multiLineOutputHandling MultiLineOutputHandling

	functionCount int
	currentLine   int
	currentChar   int
	currentItem   *parserItem

	indentChar  rune
	indentCount int

	functions *functions.Functions
	mutations *modifiers.Modifiers

	valueStore map[string]string
}

func NewParser(in io.Reader, out io.Writer, options ...OptionFunc) *Parser {
	values := make(map[string]string, 200)
	p := &Parser{
		in:  bufio.NewReader(in),
		out: bufio.NewWriter(out),

		maxFunctionCount:        -1,
		multiLineOutputHandling: MultilinePreserved,

		functions: functions.NewFunctions(values),
		mutations: modifiers.NewModifiers(),

		currentLine: 1,
		valueStore:  values,
	}
	for _, option := range options {
		option(p)
	}
	return p
}

func (p *Parser) Parse(ctx context.Context) error {
	var previousRune rune

	skipInitialize := 0      // if above 0 skips X characters, decrementing variable with every skip
	indentSection := true    // whether any char other than TAB or SPACE occurred on current line (set to false with first such occurrence)
	lastCharEscaped := false // whether last character was escaped

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		err := func() error {
			r, _, err := p.in.ReadRune()
			if err != nil {
				return err
			}
			defer func() {
				previousRune = r
			}()

			p.currentChar++
			if indentSection {
				indentSection = p.countIndent(r)
			}

			// newline
			if r == newlineChar {
				p.currentLine++
				p.currentChar = 0

				// reset indentation parsing
				p.indentChar = 0
				p.indentCount = 0
				indentSection = true
			}

			// beginning of string or function
			// - string like << abcd | upper >> has only inside processed and surrounding < and > are preserved, resulting in < ABCD >
			// - strings like <> are be skipped
			// - if another < is found before >, new item is initialized as a child
			if previousRune == itemStartChar && r != itemEndChar && skipInitialize == 0 {
				p.initializeItem(r)
				return nil
			}
			if skipInitialize > 0 {
				skipInitialize--
			}

			// ESCAPING - Start

			// eat \ instead of writing it to output
			if r == escapeChar {
				// if previous rune was also \ write it to output
				if previousRune == escapeChar && !lastCharEscaped {
					if err := p.writeRune(previousRune); err != nil {
						return err
					}
					lastCharEscaped = true
					return nil
				}
				lastCharEscaped = false
				return nil
			}

			// if previous rune is \ write current rune directly without any processing
			if previousRune == escapeChar && !lastCharEscaped {
				if err := p.writeRune(r); err != nil {
					return err
				}
				// do not initialize an item if current < was escaped
				if r == itemStartChar {
					skipInitialize++
				}
				return nil
			}
			// ESCAPING - End

			// eat < instead of writing it to output
			if r == itemStartChar {
				return nil // eat <
			}

			// no item is being processed, just write to output
			if p.currentItem == nil {
				if _, err := p.out.WriteRune(r); err != nil {
					return err
				}
				return nil
			}

			// end of currently processed item
			if r == itemEndChar {
				if err := p.processCurrentItem(); err != nil {
					return p.fmtErr(previousRune, r, err)
				}
				return nil
			}

			// if we are inside a function, detect section of the function declaration we are parsing
			cont, err := p.currentItem.ProcessCurrentFunctionSection(r)
			if err != nil {
				return p.fmtErr(previousRune, r, err)
			}
			if cont {
				return nil
			}

			// switch to modifier section for strings
			if r == modifierChar && p.currentItem.IsString() {
				p.currentItem.currSection = itemSectionModifiers
				p.currentItem.currModifier++
				return nil // eat |
			}

			return p.currentItem.AddRune(r)
		}()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}
	}

	return p.out.Flush()
}

// GetFunctionCalls returns amount of functions called at the time of the call.
// If no errors are returned from Parse function, it will be equal to the total amount of functions+modifiers in processed file.
func (p *Parser) GetFunctionCalls() int {
	return p.functionCount
}

// GetCurrentLine returns amount of lines parsed at the time of the call.
// If no errors are returned from Parse function, it will be equal to the line count of processed file.
func (p *Parser) GetCurrentLine() int {
	return p.currentLine
}

func (p *Parser) fmtErr(prev, curr rune, err error) error {
	meta := map[string][]string{
		"positionLine":       {strconv.Itoa(p.currentLine)},
		"positionColumn":     {strconv.Itoa(p.currentChar)},
		"positionNear":       {fmt.Sprintf("%c%c", prev, curr)},
		"functionCalls":      {strconv.Itoa(p.functionCount)},
		"functionCallsLimit": {strconv.Itoa(p.maxFunctionCount)},
	}

	if p.currentItem != nil {
		meta["item"] = []string{p.currentItem.name}
		meta["itemType"] = []string{p.currentItem.t.String()}
		if len(p.currentItem.parameters) != 0 {
			meta["itemParams"] = p.currentItem.GetParameters()
		}
	}

	return metaError.NewMetaError(err, meta)
}

// counts amount of indentation characters on one line
// if other char than TAB or SPACE are encountered, false is returned
func (p *Parser) countIndent(r rune) bool {
	if r != '\t' && r != ' ' {
		return false
	}
	if p.indentChar == 0 {
		p.indentChar = r
	}
	p.indentCount++
	return true
}

// writes provided rune to
// - output if currentItem is nil
// - parameters of current item if it's a function
// - name of the current item if it's a string
func (p *Parser) writeRune(r rune) error {
	if p.currentItem == nil {
		if _, err := p.out.WriteRune(r); err != nil {
			return err
		}
		return nil
	}
	if p.currentItem.IsFunction() {
		p.currentItem.parameters[p.currentItem.currParam].value += string(r)
	} else {
		p.currentItem.name += string(r)
	}

	return nil
}

// initializes a new item
// if currentItem already exists, it's set as a parent of new item
func (p *Parser) initializeItem(r rune) {
	// if rune is set to an escape char or item start char, do not pass it to constructor
	// we know it will be used to escape the following character or initialize a new item, and do not want it to be in the item's name
	if r == escapeChar || r == itemStartChar {
		r = 0
	}
	p.currentItem = newParserItem(r, p.currentItem, p.indentChar, p.indentCount)
}

// Processes currentItem.
// If it's itemTypeFunction, underlying function is called, otherwise "name" of the currentItem is used as output,
// which is then run through all provided modifyFunc and written to
//  1. output if parent of currentItem is nil
//     - if currentItem is itemTypeFunction, all newlines have indentation adjusted
//     to be the same as the beginning of the line the function was declared on
//  2. current parameter of parent of currentItem if it's a itemTypeFunction
//  3. name of the parent of currentItem if it's a itemTypeString
//
// currentItem is set to nil if it has no parent, or to the parent
func (p *Parser) processCurrentItem() error {
	if p.currentItem == nil {
		return nil
	}

	out := ""
	switch p.currentItem.t {
	case itemTypeFunction:
		if err := p.incrementFunctionCount(); err != nil {
			return err
		}

		params, err := p.currentItem.GetInterpretedParameters(p.valueStore)
		if err != nil {
			return err
		}

		out, err = p.functions.Call(p.currentItem.name, params...)
		if err != nil {
			return err
		}
	case itemTypeString:
		out = p.currentItem.name
	default:
		return fmt.Errorf("unsupported item type [%d]", p.currentItem.t) // this should never happen
	}

	for _, modifier := range p.currentItem.GetModifiers() {
		if err := p.incrementFunctionCount(); err != nil {
			return err
		}
		var err error
		out, err = p.mutations.Call(modifier, out)
		if err != nil {
			return err
		}
	}

	// handle newlines for function output (do not touch user entered text)
	if p.currentItem.t == itemTypeFunction {
		out = p.handleMultiline(out)
	}

	// if parent exists, write output to the parameters and move back to processing of the parent
	if p.currentItem.parent != nil {
		p.currentItem = p.currentItem.parent
		if p.currentItem.IsFunction() {
			p.currentItem.parameters[p.currentItem.currParam].value += out
			p.currentItem.parameters[p.currentItem.currParam].isVariable = false
		} else {
			p.currentItem.name += out
		}

		return nil
	}

	p.currentItem = nil

	if _, err := p.out.WriteString(out); err != nil {
		return err
	}
	return nil
}

func (p *Parser) handleMultiline(in string) string {
	switch p.multiLineOutputHandling {
	case MultilinePreserved:
		// do nothing
	case MultilineToOneLine:
		// collapse all newlines into single line with newlines escaped
		return strings.ReplaceAll(in, "\n", "\\n")
	case MultilineWithIndent:
		// nothing to indent, function call was first thing on the line
		if p.currentItem.indentChar == 0 {
			return in
		}
		// indent every newline to same level as the line with function definition
		return strings.ReplaceAll(in, "\n", "\n"+strings.Repeat(string(p.currentItem.indentChar), p.currentItem.indentCount))
	}
	return in
}

// increments functionCount counter and returns error if it exceeds maxFunctionCount
func (p *Parser) incrementFunctionCount() error {
	p.functionCount++
	if p.maxFunctionCount >= 0 && p.functionCount > p.maxFunctionCount {
		return fmt.Errorf("max amount of function calls [%d] exceeded", p.maxFunctionCount)
	}
	return nil
}
