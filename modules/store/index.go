package store

import (
	"fmt"
	"strings"
	"unicode"
)

// tableIndex holds all in-memory indexes for one table.
// The primary index is always populated. Secondary and text indexes
// are populated only after Schema() is called on the corresponding Model.
type tableIndex struct {
	// primary: id → disk location
	primary map[string]location

	// secondary: field → fieldValue → set of ids
	secondary  map[string]map[string]map[string]bool
	secReverse map[string]map[string]string // id → field → value (for removal)

	// text: term → set of ids
	text       map[string]map[string]bool
	txtReverse map[string][]string // id → terms (for removal)

	// schema hints
	indexedFields map[string]bool
	searchFields  map[string]bool
}

func newTableIndex() *tableIndex {
	return &tableIndex{
		primary:       make(map[string]location),
		secondary:     make(map[string]map[string]map[string]bool),
		secReverse:    make(map[string]map[string]string),
		text:          make(map[string]map[string]bool),
		txtReverse:    make(map[string][]string),
		indexedFields: make(map[string]bool),
		searchFields:  make(map[string]bool),
	}
}

// ---- primary index ----

func (idx *tableIndex) setPrimary(id string, loc location) {
	idx.primary[id] = loc
}

func (idx *tableIndex) getPrimary(id string) (location, bool) {
	loc, ok := idx.primary[id]
	return loc, ok
}

func (idx *tableIndex) deletePrimary(id string) {
	delete(idx.primary, id)
}

// allPrimaryIDs returns a snapshot of all live IDs.
func (idx *tableIndex) allPrimaryIDs() []string {
	ids := make([]string, 0, len(idx.primary))
	for id := range idx.primary {
		ids = append(ids, id)
	}
	return ids
}

// ---- secondary index ----

// updateSecondary removes oldData entries and adds newData entries for id.
// Pass nil for oldData when there is no previous version.
func (idx *tableIndex) updateSecondary(id string, oldData, newData map[string]any) {
	if len(idx.indexedFields) == 0 {
		return
	}
	// remove old
	if oldPrev, ok := idx.secReverse[id]; ok {
		for field, val := range oldPrev {
			if vals, ok := idx.secondary[field]; ok {
				if set, ok := vals[val]; ok {
					delete(set, id)
					if len(set) == 0 {
						delete(vals, val)
					}
				}
			}
		}
		delete(idx.secReverse, id)
	}
	// add new
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
		if _, ok := idx.secondary[field]; !ok {
			idx.secondary[field] = make(map[string]map[string]bool)
		}
		if _, ok := idx.secondary[field][val]; !ok {
			idx.secondary[field][val] = make(map[string]bool)
		}
		idx.secondary[field][val][id] = true
		rev[field] = val
	}
	if len(rev) > 0 {
		idx.secReverse[id] = rev
	}
}

func (idx *tableIndex) removeSecondary(id string) {
	idx.updateSecondary(id, nil, nil)
}

// lookupSecondary returns IDs matching field == value.
func (idx *tableIndex) lookupSecondary(field, value string) []string {
	vals, ok := idx.secondary[field]
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

// ---- text index ----

func (idx *tableIndex) updateText(id string, oldData, newData map[string]any) {
	if len(idx.searchFields) == 0 {
		return
	}
	// remove old terms
	if terms, ok := idx.txtReverse[id]; ok {
		for _, term := range terms {
			if set, ok := idx.text[term]; ok {
				delete(set, id)
				if len(set) == 0 {
					delete(idx.text, term)
				}
			}
		}
		delete(idx.txtReverse, id)
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
			if _, ok := idx.text[term]; !ok {
				idx.text[term] = make(map[string]bool)
			}
			idx.text[term][id] = true
		}
	}
	if len(terms) > 0 {
		idx.txtReverse[id] = terms
	}
}

func (idx *tableIndex) removeText(id string) {
	idx.updateText(id, nil, nil)
}

// searchText returns IDs that match ALL terms in the query (AND semantics).
func (idx *tableIndex) searchText(query string) []string {
	terms := tokenize(query)
	if len(terms) == 0 {
		return nil
	}
	// start with the set for the first term
	first := idx.text[terms[0]]
	if len(first) == 0 {
		return nil
	}
	result := make(map[string]bool, len(first))
	for id := range first {
		result[id] = true
	}
	// intersect with remaining terms
	for _, term := range terms[1:] {
		set := idx.text[term]
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

// ---- schema ----

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

// ---- helpers ----

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
		return strings.TrimSuffix(strings.TrimPrefix(fmt.Sprint(v), "<nil>"), "")
	}
}

// tokenize lowercases text and splits on non-alphanumeric characters.
// Tokens shorter than 2 chars are dropped.
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
