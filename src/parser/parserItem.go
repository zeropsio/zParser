package parser

import (
	"errors"
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

type parserItem struct {
	t itemType

	name              string // represents function name or string content
	parameters        []string
	paramSpacesToTrim []int // how many spaces must be trimmed of the end of parameter with same index in parameters
	modifiers         []string

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
		item.parameters = make([]string, 1, 2)
		item.paramSpacesToTrim = make([]int, 1, 2)
	} else if r != 0 {
		// if r == 0 do not add it to the name, or we will create documents with NULL bytes inside od them!
		item.name = string(r)
	}
	return item
}

// CanBeEnded returns whether item is in a section, where > should be considered as an end of the item
// This allows us the following syntax
// - <@namedString(someName, this > shouldn't be considered as an end because we are still inside parameters)>
func (i *parserItem) CanBeEnded() bool {
	return (i.IsFunction() && i.currSection == itemSectionModifiers) || i.IsString()
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
		i.parameters = append(i.parameters, "")
		i.paramSpacesToTrim = append(i.paramSpacesToTrim, 0)
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

// GetParameters returns parameters with spaces correctly trimmed
func (i *parserItem) GetParameters() []string {
	params := i.parameters
	for idx, n := range i.paramSpacesToTrim {
		params[idx] = strings.TrimRightFunc(params[idx], func(r rune) bool {
			n--
			return n >= 0
		})
	}
	return params
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
		i.parameters = append(i.parameters, "")
		i.paramSpacesToTrim = append(i.paramSpacesToTrim, 0)
	}

	// eat spaces at the beginning of the parameter
	if len(i.parameters[i.currParam]) == 0 && r == ' ' {
		return
	}

	if r == ' ' {
		i.paramSpacesToTrim[i.currParam]++
	} else {
		i.paramSpacesToTrim[i.currParam] = 0
	}
	i.parameters[i.currParam] += string(r)
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
