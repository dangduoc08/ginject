package storage

import (
	"fmt"
	"strings"
	"unicode"
)

type tableIndex struct {
	locationByID map[string]location

	secondaryByField map[string]map[string]map[string]bool
	fieldValuesByID  map[string]map[string]string

	idsByTerm map[string]map[string]bool
	termsByID map[string][]string

	indexedFields map[string]bool
	searchFields  map[string]bool
}

func newTableIndex() *tableIndex {
	return &tableIndex{
		locationByID:     make(map[string]location),
		secondaryByField: make(map[string]map[string]map[string]bool),
		fieldValuesByID:  make(map[string]map[string]string),
		idsByTerm:        make(map[string]map[string]bool),
		termsByID:        make(map[string][]string),
		indexedFields:    make(map[string]bool),
		searchFields:     make(map[string]bool),
	}
}

func (idx *tableIndex) setPrimary(id string, loc location) {
	idx.locationByID[id] = loc
}

func (idx *tableIndex) getPrimary(id string) (location, bool) {
	loc, ok := idx.locationByID[id]
	return loc, ok
}

func (idx *tableIndex) deletePrimary(id string) {
	delete(idx.locationByID, id)
}

func (idx *tableIndex) allPrimaryIDs() []string {
	ids := make([]string, 0, len(idx.locationByID))
	for id := range idx.locationByID {
		ids = append(ids, id)
	}
	return ids
}

func (idx *tableIndex) updateSecondary(id string, oldData, newData map[string]any) {
	if len(idx.indexedFields) == 0 {
		return
	}

	if oldPrev, ok := idx.fieldValuesByID[id]; ok {
		for field, val := range oldPrev {
			if vals, ok := idx.secondaryByField[field]; ok {
				if set, ok := vals[val]; ok {
					delete(set, id)
					if len(set) == 0 {
						delete(vals, val)
					}
				}
			}
		}
		delete(idx.fieldValuesByID, id)
	}

	if len(newData) == 0 {
		return
	}
	rev := make(map[string]string)
	for field := range idx.indexedFields {
		v, ok := newData[field]
		if !ok {
			continue
		}
		val := anyToString(v)
		if _, ok := idx.secondaryByField[field]; !ok {
			idx.secondaryByField[field] = make(map[string]map[string]bool)
		}
		if _, ok := idx.secondaryByField[field][val]; !ok {
			idx.secondaryByField[field][val] = make(map[string]bool)
		}
		idx.secondaryByField[field][val][id] = true
		rev[field] = val
	}
	if len(rev) > 0 {
		idx.fieldValuesByID[id] = rev
	}
}

func (idx *tableIndex) removeSecondary(id string) {
	idx.updateSecondary(id, nil, nil)
}

func (idx *tableIndex) lookupSecondary(field, value string) []string {
	vals, ok := idx.secondaryByField[field]
	if !ok {
		return nil
	}
	set, ok := vals[value]
	if !ok {
		return nil
	}
	ids := make([]string, 0, len(set))
	for id := range set {
		ids = append(ids, id)
	}
	return ids
}

func (idx *tableIndex) hasSecondaryField(field string) bool {
	return idx.indexedFields[field]
}

func (idx *tableIndex) updateText(id string, oldData, newData map[string]any) {
	if len(idx.searchFields) == 0 {
		return
	}

	if terms, ok := idx.termsByID[id]; ok {
		for _, term := range terms {
			if set, ok := idx.idsByTerm[term]; ok {
				delete(set, id)
				if len(set) == 0 {
					delete(idx.idsByTerm, term)
				}
			}
		}
		delete(idx.termsByID, id)
	}
	if len(newData) == 0 {
		return
	}
	var terms []string
	for field := range idx.searchFields {
		v, ok := newData[field]
		if !ok {
			continue
		}
		for _, term := range tokenize(anyToString(v)) {
			terms = append(terms, term)
			if _, ok := idx.idsByTerm[term]; !ok {
				idx.idsByTerm[term] = make(map[string]bool)
			}
			idx.idsByTerm[term][id] = true
		}
	}
	if len(terms) > 0 {
		idx.termsByID[id] = terms
	}
}

func (idx *tableIndex) removeText(id string) {
	idx.updateText(id, nil, nil)
}

func (idx *tableIndex) searchText(query string) []string {
	terms := tokenize(query)
	if len(terms) == 0 {
		return nil
	}

	first := idx.idsByTerm[terms[0]]
	if len(first) == 0 {
		return nil
	}
	result := make(map[string]bool, len(first))
	for id := range first {
		result[id] = true
	}

	for _, term := range terms[1:] {
		set := idx.idsByTerm[term]
		for id := range result {
			if !set[id] {
				delete(result, id)
			}
		}
		if len(result) == 0 {
			return nil
		}
	}
	ids := make([]string, 0, len(result))
	for id := range result {
		ids = append(ids, id)
	}
	return ids
}

func (idx *tableIndex) setSchema(indexedFields, searchFields []string) {
	idx.indexedFields = make(map[string]bool, len(indexedFields))
	for _, f := range indexedFields {
		idx.indexedFields[f] = true
	}
	idx.searchFields = make(map[string]bool, len(searchFields))
	for _, f := range searchFields {
		idx.searchFields[f] = true
	}
}

func anyToString(v any) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	case []byte:
		return string(t)
	default:
		return fmt.Sprint(v)
	}
}

func tokenize(s string) []string {
	s = strings.ToLower(s)
	var tokens []string
	var buf strings.Builder
	for _, ch := range s {
		if unicode.IsLetter(ch) || unicode.IsDigit(ch) {
			buf.WriteRune(ch)
		} else {
			if buf.Len() >= 2 {
				tokens = append(tokens, buf.String())
			}
			buf.Reset()
		}
	}
	if buf.Len() >= 2 {
		tokens = append(tokens, buf.String())
	}
	return tokens
}
