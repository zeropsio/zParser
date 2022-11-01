package main

import (
	"errors"
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

type itemWrap struct {
	t itemType

	name       string // represents function name or string content
	parameters []string
	modifiers  []string

	currSection  itemSection
	currParam    int
	currModifier int

	indentChar  rune
	indentCount int

	children *itemWrap
	parent   *itemWrap
}

func newItemWrap(r rune, parent *itemWrap, indentChar rune, indentCount int) *itemWrap {
	item := &itemWrap{
		t:            itemTypeString,
		parent:       parent,
		modifiers:    make([]string, 0, 5),
		currSection:  itemSectionName,
		currModifier: -1, // start at -1, because first encounter of | increments by 1
		indentChar:   indentChar,
		indentCount:  indentCount,
	}
	if r == '$' {
		item.t = itemTypeFunction
		item.parameters = make([]string, 1, 2)
	} else {
		item.name = string(r)
	}
	return item
}

func (i *itemWrap) IsWriteString() bool {
	return i != nil && i.name == "writeString" && i.currSection == itemSectionParameters
}

// CanBeEnded returns whether item is in a section, where } should be considered as an end of the item
// This allows us the following syntax
// - {$namedString(someName, this } shouldn't be considered as an end because we are still inside parameters)}
func (i *itemWrap) CanBeEnded() bool {
	return (i.IsFunction() && i.currSection == itemSectionModifiers) || i.IsString()
}

func (i *itemWrap) IsFunction() bool {
	return i != nil && i.t == itemTypeFunction
}

func (i *itemWrap) IsString() bool {
	return i != nil && i.t == itemTypeString
}

func (i *itemWrap) ProcessCurrentFunctionSection(r rune) (bool, error) {
	if !i.IsFunction() {
		return false, nil
	}

	switch r {
	case '(':
		if i.currSection != itemSectionName {
			return false, errors.New("opening brace at incorrect place")
		}
		i.currSection = itemSectionParameters
	case ')':
		if i.currSection != itemSectionParameters {
			return false, errors.New("closing brace at incorrect place")
		}
		i.currSection = itemSectionModifiers
	case ',':
		if i.currSection != itemSectionParameters {
			return false, errors.New("comma at incorrect place")
		}
		i.currParam++
		i.parameters = append(i.parameters, "")
	case '|':
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
