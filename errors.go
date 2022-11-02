package main

import (
	"fmt"
	"sort"
	"strings"
)

type MetaError struct {
	error
	Meta map[string][]string
}

func NewMetaError(error error, meta map[string][]string) *MetaError {
	return &MetaError{error: error, Meta: meta}
}

func (e MetaError) Error() string {
	return e.error.Error()
}

func (e MetaError) GetMetaAsString() string {
	if len(e.Meta) == 0 {
		return ""
	}

	sortedKeys := make([]string, 0, len(e.Meta))
	for key, _ := range e.Meta {
		sortedKeys = append(sortedKeys, key)
	}
	sort.Strings(sortedKeys)

	out := ""
	for _, key := range sortedKeys {
		out += fmt.Sprintf("%s: [%s]\n", key, strings.Join(e.Meta[key], ", "))
	}
	return out
}

func (e MetaError) GetMeta() map[string][]string {
	return e.Meta
}

func (e MetaError) GetMetaKey(key string) ([]string, bool) {
	val, found := e.Meta[key]
	return val, found
}
