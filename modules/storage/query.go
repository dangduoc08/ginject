package storage

import "strings"

// Supported operators for Where conditions.
const (
	OpEq       = "eq"       // equal
	OpNe       = "ne"       // not equal
	OpGt       = "gt"       // greater than (string comparison)
	OpLt       = "lt"       // less than (string comparison)
	OpContains = "contains" // string contains (substring)
)

// Condition is a single filter predicate.
type Condition struct {
	Field string
	Op    string
	Value any
}

// Query is a struct-based query builder. Obtain one via Model.Find().
type Query struct {
	model      *Model
	conditions []Condition
	limit      int
	skip       int
}

func newQuery(m *Model) *Query {
	return &Query{model: m}
}

// Where appends a filter condition.
func (q *Query) Where(field, op string, value any) *Query {
	q.conditions = append(q.conditions, Condition{Field: field, Op: op, Value: value})
	return q
}

// Limit sets the maximum number of results.
func (q *Query) Limit(n int) *Query {
	q.limit = n
	return q
}

// Skip sets the number of results to skip (offset).
func (q *Query) Skip(n int) *Query {
	q.skip = n
	return q
}

// Exec executes the query and returns matching documents.
func (q *Query) Exec() ([]Document, error) {
	return q.model.execQuery(q)
}

// matchesConditions returns true if doc satisfies all conditions.
func matchesConditions(doc Document, conds []Condition) bool {
	for _, c := range conds {
		v, ok := doc.Data[c.Field]
		if !ok {
			if c.Op == OpNe {
				continue
			}
			return false
		}
		docVal := anyToString(v)
		condVal := anyToString(c.Value)
		switch c.Op {
		case OpEq:
			if docVal != condVal {
				return false
			}
		case OpNe:
			if docVal == condVal {
				return false
			}
		case OpGt:
			if docVal <= condVal {
				return false
			}
		case OpLt:
			if docVal >= condVal {
				return false
			}
		case OpContains:
			if condVal != "" && !strings.Contains(docVal, condVal) {
				return false
			}
		default:
			// unknown op: treat as non-match
			return false
		}
	}
	return true
}

