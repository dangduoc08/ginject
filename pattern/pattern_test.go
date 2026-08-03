package pattern_test

import (
	"testing"

	"github.com/dangduoc08/ginject/internal/test"
	"github.com/dangduoc08/ginject/pattern"
)

func assertKind(t *testing.T, raw string, want pattern.Kind) {
	t.Helper()
	p := pattern.NewPattern(raw)
	if p.Kind() != want {
		t.Error(test.DiffMessage(p.Kind(), want, "NewPattern("+raw+").Kind()"))
	}
}

func assertMatch(t *testing.T, patternStr, topic string, want bool) {
	t.Helper()
	p := pattern.NewPattern(patternStr)
	got := p.Match(topic)
	if got != want {
		t.Error(test.DiffMessage(got, want, "Match("+patternStr+", "+topic+")"))
	}
}

func TestNewPattern_Exact(t *testing.T) {
	assertKind(t, "user.created", pattern.KindExact)
	assertKind(t, "a.b.c.d", pattern.KindExact)
	assertKind(t, "single", pattern.KindExact)
	// ">" is no longer a wildcard token, so it parses as a literal segment.
	assertKind(t, "user.>", pattern.KindExact)
}

func TestNewPattern_Global(t *testing.T) {
	assertKind(t, "*", pattern.KindGlobal)
}

func TestNewPattern_SuffixWildcard(t *testing.T) {
	assertKind(t, "user.*", pattern.KindSuffixWildcard)
	assertKind(t, "a.b.*", pattern.KindSuffixWildcard)
	assertKind(t, "a.b.c.*", pattern.KindSuffixWildcard)
}

func TestNewPattern_SuffixWildcard_SimplePrefix(t *testing.T) {
	cases := []struct{ raw, prefix string }{
		{"user.*", "user"},
		{"a.b.*", "a.b"},
		{"a.b.c.*", "a.b.c"},
	}
	for _, c := range cases {
		p := pattern.NewPattern(c.raw)
		if p.SimplePrefix() != c.prefix {
			t.Error(test.DiffMessage(p.SimplePrefix(), c.prefix, "SimplePrefix for "+c.raw))
		}
	}
}

func TestNewPattern_Complex(t *testing.T) {
	assertKind(t, "tenant.*.user.created", pattern.KindComplex)
	assertKind(t, "*.user", pattern.KindComplex)
	assertKind(t, "tenant.*.user.*", pattern.KindComplex)
}

func TestMatch_Exact(t *testing.T) {
	assertMatch(t, "user.created", "user.created", true)
	assertMatch(t, "user.created", "user.updated", false)
	assertMatch(t, "user.created", "user.created.extra", false)
	assertMatch(t, "user.created", "user", false)
}

func TestMatch_GlobalStar(t *testing.T) {
	assertMatch(t, "*", "user.created", true)
	assertMatch(t, "*", "x", true)
	assertMatch(t, "*", "a.b.c.d.e", true)
}

// SuffixWildcard is now greedy: a trailing "*" matches one or more remaining
// segments, not just exactly one.
func TestMatch_SuffixWildcard(t *testing.T) {
	assertMatch(t, "user.*", "user.created", true)
	assertMatch(t, "user.*", "user.deleted", true)
	assertMatch(t, "user.*", "user.profile.updated", true)
	assertMatch(t, "user.*", "user.a.b.c.d", true)
	assertMatch(t, "user.*", "order.created", false)
	assertMatch(t, "user.*", "user", false)
	assertMatch(t, "user.*", "userx.created", false)

	assertMatch(t, "a.b.*", "a.b.c", true)
	assertMatch(t, "a.b.*", "a.b.c.d", true)
	assertMatch(t, "a.b.*", "a.c", false)
}

// A "*" that is not the pattern's last token keeps the original,
// single-segment-only behavior.
func TestMatch_Complex_MiddleWildcard(t *testing.T) {
	assertMatch(t, "tenant.*.user.created", "tenant.1.user.created", true)
	assertMatch(t, "tenant.*.user.created", "tenant.abc.user.created", true)
	assertMatch(t, "tenant.*.user.created", "tenant.1.user.updated", false)
	assertMatch(t, "tenant.*.user.created", "tenant.1.2.user.created", false)
	assertMatch(t, "tenant.*.user.created", "tenant.user.created", false)
}

// When a pattern has a wildcard both mid-pattern and as the trailing token,
// only the trailing one is greedy.
func TestMatch_Complex_MiddleAndTrailingWildcard(t *testing.T) {
	assertMatch(t, "tenant.*.user.*", "tenant.1.user.created", true)
	assertMatch(t, "tenant.*.user.*", "tenant.1.user.profile.updated", true)
	assertMatch(t, "tenant.*.user.*", "tenant.abc.user.settings.avatar.changed", true)
	assertMatch(t, "tenant.*.user.*", "tenant.1.admin.created", false)
	assertMatch(t, "tenant.*.user.*", "tenant.1.user", false)
}

func TestMatch_Complex_LeadingWildcard(t *testing.T) {
	assertMatch(t, "*.created", "user.created", true)
	assertMatch(t, "*.created", "order.created", true)
	assertMatch(t, "*.created", "user.updated", false)
	assertMatch(t, "*.created", "a.b.created", false)
}

func TestMatch_BackwardCompat(t *testing.T) {
	assertMatch(t, "*", "anything", true)
	assertMatch(t, "*", "a.b.c", true)
	assertMatch(t, "user.*", "user.x", true)
	assertMatch(t, "user.*", "user.x.y", true)
}
