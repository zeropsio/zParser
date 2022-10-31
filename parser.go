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

	functionCount int
	currentLine   int
	currentItem   *itemWrap

	functions *Functions
	mutations *Modifiers
}

func NewImportParser(in io.Reader, out io.Writer) *ImportParser {
	p := &ImportParser{
		in:  bufio.NewReader(in),
		out: bufio.NewWriter(out),

		functions: NewFunctions(),
		mutations: NewModifiers(),

		currentLine: 1,
	}

	return p
}

func (p *ImportParser) Parse() error {
	var previousRune rune
	skipItemCount := 0
	for {
		err := func() error {
			r, _, err := p.in.ReadRune()
			if err != nil {
				return err
			}
			defer func() {
				previousRune = r
			}()
			if r == 0x000A {
				p.currentLine++
			}

			// as long as we are in "writeString" function (which acts as a pass through), blindly accept everything up to first )
			if p.currentItem.IsWriteString() && r != ')' {
				p.currentItem.parameters[p.currentItem.currParam] += string(r)
				return nil
			}

			// ignore env variables with following syntax: ${my_env}
			if r == '{' && previousRune == '$' {
				if p.currentItem.IsFunction() {
					return p.fmtErr(previousRune, r, errors.New("env syntax `${xxx}` is not allowed inside function parameters"))
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
				// eat {
				return nil
			}

			// beginning of string or function
			// - string like {{ abcd | upper }} has only inside processed and surrounding { and } are preserved, resulting in { ABCD }
			// - strings like {} are be skipped
			// - if another { is found before }, new item is initialized as a child
			if previousRune == '{' && r != '{' && r != '}' {
				p.initializeItem(r)
				return nil
			}

			// end of currently processed item
			// TODO(ms): refactor
			if r == '}' && ((p.currentItem.IsFunction() && p.currentItem.currSection == itemSectionModifiers) || p.currentItem.IsString()) {
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

			if p.currentItem.IsFunction() {
				// if we are inside a function, detect section of the function declaration we are parsing
				cont, err := p.currentItem.ProcessCurrentFunctionSection(r)
				if err != nil {
					return p.fmtErr(previousRune, r, err)
				}
				if cont {
					return nil
				}

				// TODO(ms): refactor
				if p.currentItem.currSection == itemSectionModifiers && p.currentItem.currModifier == -1 {
					if r != ' ' {
						return p.fmtErr(previousRune, r, errors.New("invalid character, expected space od modifier character"))
					} else {
						return nil
					}
				}
			} else {
				// switch to modifier section
				// - after function section detection, this way a pipe is allowed inside a function parameter
				if r == '|' {
					// eat |
					p.currentItem.currSection = itemSectionModifiers
					p.currentItem.currModifier++
					return nil
				}
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
	paramStr := ""
	for i, parameter := range p.currentItem.parameters {
		paramStr += fmt.Sprintf("\nparam %d: `%s`", i+1, parameter)
	}
	return fmt.Errorf("error: %w\nline: %d\nnear: `%c%c`\nprocessing: `%s`%s", err, p.currentLine, prev, curr, p.currentItem.name, paramStr)
}

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

func (p *ImportParser) initializeItem(r rune) {
	item := newItemWrap(r, p.currentItem)
	if p.currentItem != nil {
		item.parent = p.currentItem
	}
	p.currentItem = item
}

func (p *ImportParser) processCurrentItem() error {
	if p.currentItem == nil {
		return nil
	}

	p.currentItem.name = strings.TrimSpace(p.currentItem.name)

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
	case itemTypeString:
		out = p.currentItem.name
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

	// if no parent exist, clear currentItem and write string to output
	p.currentItem = nil

	if _, err := p.out.WriteString(out); err != nil {
		return err
	}

	return nil
}

func (p *ImportParser) incrementFunctionCount() error {
	const maxFunctionCalls = 200
	p.functionCount++
	if p.functionCount > maxFunctionCalls {
		return fmt.Errorf("max amount of function calls [%d] exceeded", maxFunctionCalls)
	}
	return nil
}
