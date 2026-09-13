package storage

import "strings"

const (
	OpEq       = "eq"
	OpNe       = "ne"
	OpGt       = "gt"
	OpLt       = "lt"
	OpContains = "contains"
)

type Condition struct {
	Field string
	Op    string
	Value any
}

type Query struct {
	model      *Model
	conditions []Condition
	limit      int
	skip       int
}

func newQuery(m *Model) *Query {
	return &Query{model: m}
}

func (q *Query) Where(field, op string, value any) *Query {
	q.conditions = append(q.conditions, Condition{Field: field, Op: op, Value: value})
	return q
}

func (q *Query) Limit(n int) *Query {
	q.limit = n
	return q
}

func (q *Query) Skip(n int) *Query {
	q.skip = n
	return q
}

func (q *Query) Exec() ([]Document, error) {
	return q.model.execQuery(q)
}

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

			return false
		}
	}
	return true
}
