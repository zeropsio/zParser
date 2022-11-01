package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"
)

type ImportParser struct {
	out *bufio.Writer
	in  *bufio.Reader

	maxFunctionCount int
	functionCount    int

	currentLine int
	currentItem *itemWrap

	indentChar  rune
	indentCount int

	functions *Functions
	mutations *Modifiers
}

func NewImportParser(in io.Reader, out io.Writer, maxFuncCount int) *ImportParser {
	p := &ImportParser{
		in:  bufio.NewReader(in),
		out: bufio.NewWriter(out),

		maxFunctionCount: maxFuncCount,

		functions: NewFunctions(),
		mutations: NewModifiers(),

		currentLine: 1,
	}

	return p
}

func (p *ImportParser) Parse() error {
	var previousRune rune
	skipItemCount := 0
	indentSection := true // whether we are still at the beginning of the line, counting tabs/spaces for size of indentation

	for {
		err := func() error {
			r, _, err := p.in.ReadRune()
			if err != nil {
				return err
			}
			defer func() {
				previousRune = r
			}()
			if indentSection {
				indentSection = p.countIndent(r)
			}
			// newline
			if r == 0x000A {
				p.currentLine++
				// reset indent counting
				p.indentChar = 0x0000
				p.indentCount = 0
				indentSection = true
			}

			// TODO(ms): clean up
			// escaping -> eat \ instead of writing it to output
			if r == '\\' {
				// if previous rune was also \ write it to output
				if previousRune == '\\' {
					if err := p.writeRune(previousRune); err != nil {
						return err
					}
				}
				return nil // eat \
			}
			// escaping if previous rune is \ write current rune directly without any processing
			if previousRune == '\\' {
				if err := p.writeRune(r); err != nil {
					return err
				}
				return nil
			}

			// as long as we are in "writeString" function (which acts as a pass through), blindly accept everything up to first )
			if p.currentItem.IsWriteString() && r != ')' {
				p.currentItem.parameters[p.currentItem.currParam] += string(r)
				return nil
			}

			// ignore env variables with following syntax: ${my_env}
			if r == '{' && previousRune == '$' {
				if p.currentItem.IsFunction() {
					return p.fmtErr(previousRune, r, errors.New("env syntax `${xxx}` is not allowed inside function call"))
				}
				skipItemCount++
			}

			if skipItemCount > 0 {
				if err := p.writeRune(r); err != nil {
					return err
				}
				if r == '}' {
					skipItemCount--
				}
				return nil
			}

			// eat { instead of writing it to output
			if r == '{' {
				// if previous rune was { write it to output (we need to eat only last occurrence of { in a chain)
				if previousRune == '{' {
					if err := p.writeRune(previousRune); err != nil {
						return err
					}
				}
				return nil // eat {
			}

			// beginning of string or function
			// - string like {{ abcd | upper }} has only inside processed and surrounding { and } are preserved, resulting in { ABCD }
			// - strings like {} are be skipped
			// - if another { is found before }, new item is initialized as a child
			if previousRune == '{' && r != '{' && r != '}' {
				p.initializeItem(r)
				return nil
			}

			if previousRune == '{' && r == '}' {
				return nil // eat {}
			}

			// end of currently processed item
			if r == '}' && p.currentItem.CanBeEnded() {
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
			if r == '|' && p.currentItem.IsString() {
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

func (p *ImportParser) fmtErr(prev, curr rune, err error) error {
	// TODO(ms): use meta errors instead of newlines
	paramStr := ""
	for i, parameter := range p.currentItem.parameters {
		paramStr += fmt.Sprintf("\nparam %d: `%s`", i+1, parameter)
	}
	return fmt.Errorf("error: %w\nline: %d\nnear: `%c%c`\nprocessing: `%s`%s", err, p.currentLine, prev, curr, p.currentItem.name, paramStr)
}

// counts amount of indentation characters on one line
// if other char than TAB or SPACE are encountered, false is returned
func (p *ImportParser) countIndent(r rune) bool {
	if r != 0x0009 && r != ' ' {
		return false
	}
	if p.indentChar == 0x0000 {
		p.indentChar = r
	}
	p.indentCount++
	return true
}

// writes provided rune to
// - output if currentItem is nil
// - parameters of current item if it's a function
// - name of the current item if it's a string
func (p *ImportParser) writeRune(r rune) error {
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
func (p *ImportParser) initializeItem(r rune) {
	item := newItemWrap(r, p.currentItem, p.indentChar, p.indentCount)
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
func (p *ImportParser) processCurrentItem() error {
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
	if addIndent && p.currentItem.indentChar != 0x0000 {
		out = strings.ReplaceAll(out, "\n", "\n"+strings.Repeat(string(p.currentItem.indentChar), p.currentItem.indentCount))
	}

	p.currentItem = nil

	if _, err := p.out.WriteString(out); err != nil {
		return err
	}
	return nil
}

// increments functionCount counter and returns error if it exceeds maxFunctionCount
func (p *ImportParser) incrementFunctionCount() error {
	p.functionCount++
	if p.functionCount > p.maxFunctionCount {
		return fmt.Errorf("max amount of function calls [%d] exceeded", p.maxFunctionCount)
	}
	return nil
}
