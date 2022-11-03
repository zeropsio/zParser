package parser

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"git.vsh-labs.cz/zerops/yaml-parser/src/functions"
	"git.vsh-labs.cz/zerops/yaml-parser/src/metaError"
	"git.vsh-labs.cz/zerops/yaml-parser/src/modifiers"
)

const (
	escapeChar     = '\\'
	newlineChar    = '\n'
	itemStartChar  = '{'
	itemEndChar    = '}'
	funcStartChar  = '$' // combined with itemStartChar which must be preceding funcStartChar
	modifierChar   = '|'
	paramStartChar = '('
	paramEndChar   = ')'
	paramSepChar   = ','
)

type Parser struct {
	out *bufio.Writer
	in  *bufio.Reader

	maxFunctionCount int
	functionCount    int

	currentLine int
	currentChar int
	currentItem *parserItem

	indentChar  rune
	indentCount int

	functions *functions.Functions
	mutations *modifiers.Modifiers
}

func NewParser(in io.Reader, out io.Writer, maxFuncCount int) *Parser {
	p := &Parser{
		in:  bufio.NewReader(in),
		out: bufio.NewWriter(out),

		maxFunctionCount: maxFuncCount,

		functions: functions.NewFunctions(),
		mutations: modifiers.NewModifiers(),

		currentLine: 1,
	}

	return p
}

func (p *Parser) Parse() error {
	var previousRune rune

	skipInitialize := 0      // if above 0 skips X characters, decrementing variable with every skip
	skipEnv := false         // if set to true, skips all characters until first unescaped } and sets variable to false
	indentSection := true    // whether any char other than TAB or SPACE occurred on current line (set to false with first such occurrence)
	lastCharEscaped := false // whether last character was escaped

	for {
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
			// - string like {{ abcd | upper }} has only inside processed and surrounding { and } are preserved, resulting in { ABCD }
			// - strings like {} are be skipped
			// - if another { is found before }, new item is initialized as a child
			if previousRune == itemStartChar && r != itemEndChar && !skipEnv && skipInitialize == 0 {
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
				// lastCharEscaped = false
				if err := p.writeRune(r); err != nil {
					return err
				}
				// do not initialize an item if current { was escaped
				if r == itemStartChar {
					skipInitialize++
				}
				return nil
			}

			// ignore env variables with following syntax: ${my_env}
			if r == itemStartChar && previousRune == funcStartChar {
				if p.currentItem.IsFunction() {
					return p.fmtErr(previousRune, r, errors.New("env syntax `${xxx}` is not allowed inside function call"))
				}
				skipEnv = true
			}

			if skipEnv {
				if err := p.writeRune(r); err != nil {
					return err
				}
				if r == itemEndChar {
					skipEnv = false
				}
				return nil
			}
			// ESCAPING - End

			// eat { instead of writing it to output
			if r == itemStartChar {
				return nil // eat {
			}

			// end of currently processed item
			if r == itemEndChar && p.currentItem.CanBeEnded() {
				if err := p.processCurrentItem(); err != nil {
					return p.fmtErr(previousRune, r, err)
				}
				return nil
			}

			// no item is being processed, just write to output
			if p.currentItem == nil {
				if _, err := p.out.WriteRune(r); err != nil {
					return err
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

			switch p.currentItem.currSection {
			case itemSectionName:
				p.currentItem.name += string(r)
			case itemSectionParameters:
				if len(p.currentItem.parameters) < p.currentItem.currParam+1 {
					p.currentItem.parameters = append(p.currentItem.parameters, string(r))
				} else {
					p.currentItem.parameters[p.currentItem.currParam] += string(r)
				}
			case itemSectionModifiers:
				// this prevents issues with spaces between function closing parentheses and first |
				if p.currentItem.currModifier == -1 {
					return nil
				}
				if len(p.currentItem.modifiers) < p.currentItem.currModifier+1 {
					p.currentItem.modifiers = append(p.currentItem.modifiers, string(r))
				} else {
					p.currentItem.modifiers[p.currentItem.currModifier] += string(r)
				}
			}

			return nil
		}()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}

	return p.out.Flush()
}

func (p *Parser) fmtErr(prev, curr rune, err error) error {
	meta := map[string][]string{
		"line":             {strconv.Itoa(p.currentLine)},
		"column":           {strconv.Itoa(p.currentChar)},
		"near":             {fmt.Sprintf("%c%c", prev, curr)},
		"functionCalls":    {strconv.Itoa(p.functionCount)},
		"maxFunctionCalls": {strconv.Itoa(p.maxFunctionCount)},
	}

	if p.currentItem != nil {
		meta["processing"] = []string{p.currentItem.name}
		for i, parameter := range p.currentItem.parameters {
			meta["param"+strconv.Itoa(i+1)] = []string{parameter}
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
		p.currentItem.parameters[p.currentItem.currParam] += string(r)
	} else {
		p.currentItem.name += string(r)
	}

	return nil
}

// initializes a new item
// if currentItem already exists, it's set as a parent of new item
func (p *Parser) initializeItem(r rune) {
	// if rune is set to an escape char, do not pass it to constructor
	// we know it will be used to escape the following character, and do not want it to be in the item's name
	if r == escapeChar {
		r = 0
	}
	item := newParserItem(r, p.currentItem, p.indentChar, p.indentCount)
	if p.currentItem != nil {
		item.parent = p.currentItem
	}
	p.currentItem = item
}

// Processes currentItem
//
// If it's itemTypeFunction, underlying function is called, otherwise "name" of the currentItem is used as output,
// which is then run through all provided modifyFunc and written to
//   - output if parent of currentItem is nil
//     - if currentItem is itemTypeFunction, all newlines have indentation adjusted
//       to be the same as the beginning of the line the function was declared on
//   - current parameter of parent of currentItem if it's a itemTypeFunction
//   - name of the parent of currentItem if it's a itemTypeString
// currentItem is set to nil if it has no parent, or to the parent
func (p *Parser) processCurrentItem() error {
	if p.currentItem == nil {
		return nil
	}

	p.currentItem.name = strings.TrimSpace(p.currentItem.name)

	addIndent := false // whether indentation should be added to the output - used only for functions (they may generate multiline output)
	out := ""
	switch p.currentItem.t {
	case itemTypeFunction:
		if err := p.incrementFunctionCount(); err != nil {
			return err
		}
		// remove spaces from parameters, otherwise spaces between commas would break things
		for i, parameter := range p.currentItem.parameters {
			p.currentItem.parameters[i] = strings.TrimSpace(parameter)
		}
		var err error
		out, err = p.functions.Call(p.currentItem.name, p.currentItem.parameters...)
		if err != nil {
			return err
		}
		addIndent = true
	case itemTypeString:
		out = p.currentItem.name
	default:
		return fmt.Errorf("unsupported item type [%d]", p.currentItem.t) // this should never happen
	}

	for _, modifier := range p.currentItem.modifiers {
		if err := p.incrementFunctionCount(); err != nil {
			return err
		}
		modifier = strings.TrimSpace(modifier)
		var err error
		out, err = p.mutations.Call(modifier, out)
		if err != nil {
			return err
		}
	}

	// if parent exists, write output to the parameters and move back to processing of the parent
	if p.currentItem.parent != nil {
		p.currentItem = p.currentItem.parent
		if p.currentItem.IsFunction() {
			p.currentItem.parameters[p.currentItem.currParam] += out
		} else {
			p.currentItem.name += out
		}

		return nil
	}

	// indent every newline to same level as the line with function definition
	if addIndent && p.currentItem.indentChar != 0 {
		out = strings.ReplaceAll(out, "\n", "\n"+strings.Repeat(string(p.currentItem.indentChar), p.currentItem.indentCount))
	}

	p.currentItem = nil

	if _, err := p.out.WriteString(out); err != nil {
		return err
	}
	return nil
}

// increments functionCount counter and returns error if it exceeds maxFunctionCount
func (p *Parser) incrementFunctionCount() error {
	p.functionCount++
	if p.functionCount > p.maxFunctionCount {
		return fmt.Errorf("max amount of function calls [%d] exceeded", p.maxFunctionCount)
	}
	return nil
}
