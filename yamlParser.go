package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
)

type ImportParser struct {
	out *bytes.Buffer // TODO(ms): use writer instead of a buffer
	in  *bufio.Reader

	functionCount int
	currentItem   *yamlParserItemWrap

	functions *YamlFunctions
	mutations *YamlModifiers
}

func NewImportParser(in io.Reader) *ImportParser {
	b := make([]byte, 0, 1024*100) // kB
	p := &ImportParser{
		in:        bufio.NewReader(in),
		out:       bytes.NewBuffer(b),
		functions: NewYamlFunctions(),
		mutations: NewYamlModifiers(),
	}

	return p
}

func (p *ImportParser) GetOutput() *bytes.Buffer {
	return p.out
}

func (p *ImportParser) Parse() error {
	var previousRune rune
	// TODO(ms): naming, might be used to just skip parsing not only env (probably for writeSting function)
	//  - probably use int that is incremented on every { and decremented on every }
	//    that should work well with writeString function
	parsingEnv := false
	for {
		r, _, err := p.in.ReadRune()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// ignore env variables with following syntax: ${my_env}
		if r == '{' && previousRune == '$' {
			if p.currentItem != nil {
				return errors.New("env syntax `${xxx}` is not allowed inside function parameters")
			}
			parsingEnv = true
		}

		if parsingEnv {
			if _, err := p.out.WriteRune(r); err != nil {
				return err
			}
			if r == '}' {
				parsingEnv = false
			}
			previousRune = r
			continue
		}

		// eat { instead of writing it to output
		if r == '{' {
			// if previous rune was { write it to output (we need to eat only last { occurrence in a chain)
			if previousRune == '{' {
				if _, err := p.out.WriteRune(previousRune); err != nil {
					return err
				}
			}
			// eat {
			previousRune = r
			continue
		}

		// beginning of string or function
		// - string like {{ abcd | upper }} has only inside processed and surrounding { and } are preserved, resulting in { ABCD }
		// - strings like {} are be skipped
		// - if another { is found before }, new item is initialized as a child
		if previousRune == '{' && r != '{' && r != '}' {
			p.initializeItem(r)
			previousRune = r
			continue
		}

		// end of currently processed item
		if r == '}' && p.currentItem != nil {
			previousRune = r
			if err := p.processCurrentItem(); err != nil {
				return err
			}
			continue
		}

		// no item is being processed, just write to output
		if p.currentItem == nil {
			if _, err := p.out.WriteRune(r); err != nil {
				return err
			}
			previousRune = r
			continue
		}

		if p.currentItem.IsFunction() {
			// if we are inside a function, detect section of the function declaration we are parsing
			if cont, err := p.currentItem.ProcessCurrentFunctionSection(r); err != nil {
				return err
			} else if cont {
				previousRune = r
				continue
			}
		}

		// switch to modifier section
		// - after function section detection, this way a pipe is allowed inside a function parameter
		if r == '|' {
			if p.currentItem.IsFunction() && p.currentItem.currSection == yamlItemSectionName {
				return errors.New("modifier character is not allowed in a function name")
			}
			// eat |
			p.currentItem.currSection = yamlItemSectionModifiers
			p.currentItem.currModifier++
			previousRune = r
			continue
		}

		switch p.currentItem.currSection {
		case yamlItemSectionName:
			p.currentItem.name += string(r)
		case yamlItemSectionParameters:
			if len(p.currentItem.parameters) < p.currentItem.currParam+1 {
				p.currentItem.parameters = append(p.currentItem.parameters, string(r))
			} else {
				p.currentItem.parameters[p.currentItem.currParam] += string(r)
			}
		case yamlItemSectionModifiers:
			if len(p.currentItem.modifiers) < p.currentItem.currModifier+1 {
				p.currentItem.modifiers = append(p.currentItem.modifiers, string(r))
			} else {
				p.currentItem.modifiers[p.currentItem.currModifier] += string(r)
			}
		}

		previousRune = r
	}

	return nil
}

func (p *ImportParser) initializeItem(r rune) {
	item := newYamlParserItemWrap(r, p.currentItem)
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
	case yamlItemTypeFunction:
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
	case yamlItemTypeString:
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
			p.currentItem.parameters[p.currentItem.currParam] = out
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
