package parser

import (
	"errors"
	"fmt"
	"strings"
)

type itemType int
type itemSection int

const (
	itemTypeFunction itemType = iota
	itemTypeString
)
const (
	itemSectionName itemSection = iota // function name for name itemTypeFunction or string content for itemTypeString
	itemSectionParameters
	itemSectionModifiers
)

func (t itemType) String() string {
	switch t {
	case itemTypeFunction:
		return "function"
	case itemTypeString:
		return "string"
	}
	return "unknown"
}

type itemParam struct {
	value      string
	isVariable bool
}

type parserItem struct {
	t itemType

	name       string // represents function name or string content
	parameters []itemParam
	modifiers  []string

	currSection  itemSection
	currParam    int
	currModifier int

	indentChar  rune
	indentCount int

	parent *parserItem
}

func newParserItem(r rune, parent *parserItem, indentChar rune, indentCount int) *parserItem {
	item := &parserItem{
		t:            itemTypeString,
		parent:       parent,
		modifiers:    make([]string, 0, 5),
		currSection:  itemSectionName,
		currModifier: -1, // start at -1, because first encounter of | increments by 1
		indentChar:   indentChar,
		indentCount:  indentCount,
	}
	if r == funcStartChar {
		item.t = itemTypeFunction
		item.parameters = []itemParam{{
			value:      "",
			isVariable: true,
		}}
	} else if r != 0 {
		// if r == 0 do not add it to the name, otherwise we would create documents with NULL bytes inside!
		item.name = string(r)
	}
	return item
}

func (i *parserItem) IsFunction() bool {
	return i != nil && i.t == itemTypeFunction
}

func (i *parserItem) IsString() bool {
	return i != nil && i.t == itemTypeString
}

func (i *parserItem) ProcessCurrentFunctionSection(r rune) (bool, error) {
	if !i.IsFunction() {
		return false, nil
	}

	switch r {
	case paramStartChar:
		if i.currSection != itemSectionName {
			return false, errors.New("opening brace at incorrect place")
		}
		i.currSection = itemSectionParameters
	case paramEndChar:
		if i.currSection != itemSectionParameters {
			return false, errors.New("closing brace at incorrect place")
		}
		i.currSection = itemSectionModifiers
	case paramSepChar:
		if i.currSection != itemSectionParameters {
			return false, errors.New("comma at incorrect place")
		}
		i.currParam++
	case modifierChar:
		if i.currSection == itemSectionName {
			return false, errors.New("modifier character is not allowed in a function name")
		} else if i.currSection == itemSectionParameters {
			return false, nil // allow | inside parameters section
		}
		i.currModifier++
	case ' ': // eat spaces between function closing brace and first |
		if i.currSection != itemSectionModifiers || i.currModifier != -1 {
			return false, nil
		}
	default: // continue parsing
		// validate only spaces are between function closing brace and first |
		if i.currSection == itemSectionModifiers && i.currModifier == -1 {
			return false, errors.New("invalid character, expected space od modifier character")
		}
		return false, nil
	}
	return true, nil // eat current rune
}

func (i *parserItem) AddRune(r rune) error {
	switch i.currSection {
	case itemSectionName:
		i.name += string(r)
	case itemSectionParameters:
		i.addToParameter(r)
	case itemSectionModifiers:
		i.addToModifier(r)
	}
	return nil
}

// GetParameters returns plain parameters (with variable names not interpreted) with spaces correctly trimmed
func (i *parserItem) GetParameters() []string {
	params := make([]string, len(i.parameters))
	for idx, param := range i.parameters {
		if !param.isVariable {
			params[idx] = param.value
			continue
		}
		params[idx] = strings.TrimSpace(param.value)
	}
	return params
}

// GetInterpretedParameters returns parameters with variables interpreted
// TODO(ms): find a better way than passing valueStore in
func (i *parserItem) GetInterpretedParameters(valueStore map[string]string) ([]string, error) {
	params := make([]string, len(i.parameters))
	for idx, param := range i.parameters {
		if !param.isVariable {
			params[idx] = param.value
			continue
		}

		value := strings.TrimSpace(param.value)
		val, found := valueStore[value]
		if !found {
			return nil, fmt.Errorf("variable [%s] not found", value)
		}

		params[idx] = val
	}
	return params, nil
}

// GetModifiers returns modifiers with spaces trimmed
func (i *parserItem) GetModifiers() []string {
	modifiers := i.modifiers
	for idx, modifier := range modifiers {
		modifiers[idx] = strings.TrimSpace(modifier)
	}
	return modifiers
}

// adds rune to current parameter
func (i *parserItem) addToParameter(r rune) {
	// initializes parameter
	if len(i.parameters) < i.currParam+1 {
		i.parameters = append(i.parameters, itemParam{
			value:      "",
			isVariable: true,
		})
	}

	// eat spaces at the beginning of the parameter
	if len(i.parameters[i.currParam].value) == 0 && r == ' ' {
		return
	}
	if r != ' ' {
		i.parameters[i.currParam].isVariable = true
	}
	i.parameters[i.currParam].value += string(r)
}

// adds rune to current modifier
func (i *parserItem) addToModifier(r rune) {
	// this prevents issues with spaces between function closing parentheses and first |
	if i.currModifier == -1 {
		return
	}
	if len(i.modifiers) < i.currModifier+1 {
		i.modifiers = append(i.modifiers, string(r))
	} else {
		i.modifiers[i.currModifier] += string(r)
	}
}
