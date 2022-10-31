package main

import (
	"errors"
)

type YamlParserItemType int
type YamlParserItemSection int

const (
	yamlItemTypeFunction YamlParserItemType = iota
	yamlItemTypeString
)
const (
	yamlItemSectionName YamlParserItemSection = iota // function name for name yamlItemTypeFunction or string content for yamlItemTypeString
	yamlItemSectionParameters
	yamlItemSectionModifiers
)

type yamlParserItemWrap struct {
	t YamlParserItemType

	name       string // represents function name or string content
	parameters []string
	modifiers  []string

	currSection  YamlParserItemSection
	currParam    int
	currModifier int

	children *yamlParserItemWrap
	parent   *yamlParserItemWrap
}

func newYamlParserItemWrap(r rune, parent *yamlParserItemWrap) *yamlParserItemWrap {
	item := &yamlParserItemWrap{
		t:            yamlItemTypeString,
		parent:       parent,
		modifiers:    make([]string, 0, 5),
		currSection:  yamlItemSectionName,
		currModifier: -1, // start at -1, because first encounter of | increments by 1
	}
	if r == '$' {
		item.t = yamlItemTypeFunction
		item.parameters = make([]string, 1, 2)
	} else {
		item.name = string(r)
	}
	return item
}

func (i *yamlParserItemWrap) IsWriteString() bool {
	return i != nil && i.name == "writeString" && i.currSection == yamlItemSectionParameters
}

func (i *yamlParserItemWrap) IsFunction() bool {
	return i != nil && i.t == yamlItemTypeFunction
}

func (i *yamlParserItemWrap) IsString() bool {
	return i != nil && i.t == yamlItemTypeString
}

func (i *yamlParserItemWrap) ProcessCurrentFunctionSection(r rune) (bool, error) {
	switch r {
	case '|':
		if i.currSection == yamlItemSectionName {
			return false, errors.New("modifier character is not allowed in a function name")
		} else if i.currSection == yamlItemSectionParameters {
			return false, nil
		}
		i.currModifier++
	case '(':
		// eat (
		if i.currSection != yamlItemSectionName {
			return false, errors.New("opening brace at incorrect place")
		}
		i.currSection = yamlItemSectionParameters
	case ')':
		// eat )
		if i.currSection != yamlItemSectionParameters {
			return false, errors.New("closing brace at incorrect place")
		}
		i.currSection = yamlItemSectionModifiers
	case ',':
		if i.currSection != yamlItemSectionParameters {
			return false, errors.New("comma at incorrect place")
		}
		// eat ,
		i.currParam++
		i.parameters = append(i.parameters, "")
	default:
		return false, nil
	}
	return true, nil
}
