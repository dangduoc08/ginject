package matcher_test

import (
	"testing"

	"github.com/dangduoc08/ginject/matcher"
	"github.com/dangduoc08/ginject/testutils"
)

func assertKind(t *testing.T, raw string, want matcher.Kind) {
	t.Helper()
	p := matcher.Parse(raw)
	if p.Kind() != want {
		t.Error(testutils.DiffMessage(p.Kind(), want, "Parse("+raw+").Kind()"))
	}
}

func assertMatch(t *testing.T, pattern, topic string, want bool) {
	t.Helper()
	p := matcher.Parse(pattern)
	got := matcher.Match(p, topic)
	if got != want {
		t.Error(testutils.DiffMessage(got, want, "Match("+pattern+", "+topic+")"))
	}
}

func TestParse_Exact(t *testing.T) {
	assertKind(t, "user.created", matcher.KindExact)
	assertKind(t, "a.b.c.d", matcher.KindExact)
	assertKind(t, "single", matcher.KindExact)
}

func TestParse_Global(t *testing.T) {
	assertKind(t, "*", matcher.KindGlobal)
	assertKind(t, ">", matcher.KindGlobal)
}

func TestParse_SingleSuffix(t *testing.T) {
	assertKind(t, "user.*", matcher.KindSingleSuffix)
	assertKind(t, "a.b.*", matcher.KindSingleSuffix)
	assertKind(t, "a.b.c.*", matcher.KindSingleSuffix)
}

func TestParse_SingleSuffix_SimplePrefix(t *testing.T) {
	cases := []struct{ raw, prefix string }{
		{"user.*", "user"},
		{"a.b.*", "a.b"},
		{"a.b.c.*", "a.b.c"},
	}
	for _, c := range cases {
		p := matcher.Parse(c.raw)
		if p.SimplePrefix() != c.prefix {
			t.Error(testutils.DiffMessage(p.SimplePrefix(), c.prefix, "SimplePrefix for "+c.raw))
		}
	}
}

func TestParse_Complex(t *testing.T) {
	assertKind(t, "user.>", matcher.KindComplex)
	assertKind(t, ">", matcher.KindGlobal)
	assertKind(t, "a.b.>", matcher.KindComplex)
	assertKind(t, "tenant.*.user.created", matcher.KindComplex)
	assertKind(t, "tenant.*.user.>", matcher.KindComplex)
	assertKind(t, "*.user", matcher.KindComplex)
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

func TestMatch_GlobalMulti(t *testing.T) {
	assertMatch(t, ">", "user.created", true)
	assertMatch(t, ">", "x", true)
	assertMatch(t, ">", "a.b.c.d.e", true)
}

func TestMatch_SingleSuffix(t *testing.T) {
	assertMatch(t, "user.*", "user.created", true)
	assertMatch(t, "user.*", "user.deleted", true)
	assertMatch(t, "user.*", "user.profile.updated", false)
	assertMatch(t, "user.*", "order.created", false)
	assertMatch(t, "user.*", "user", false)

	assertMatch(t, "a.b.*", "a.b.c", true)
	assertMatch(t, "a.b.*", "a.b.c.d", false)
	assertMatch(t, "a.b.*", "a.c", false)
}

func TestMatch_MultiLevelSuffix(t *testing.T) {
	assertMatch(t, "user.>", "user.created", true)
	assertMatch(t, "user.>", "user.profile.updated", true)
	assertMatch(t, "user.>", "user.a.b.c.d", true)
	assertMatch(t, "user.>", "user", false)
	assertMatch(t, "user.>", "order.created", false)
	assertMatch(t, "user.>", "userx.created", false)
}

func TestMatch_Complex_MiddleWildcard(t *testing.T) {
	assertMatch(t, "tenant.*.user.created", "tenant.1.user.created", true)
	assertMatch(t, "tenant.*.user.created", "tenant.abc.user.created", true)
	assertMatch(t, "tenant.*.user.created", "tenant.1.user.updated", false)
	assertMatch(t, "tenant.*.user.created", "tenant.1.2.user.created", false)
	assertMatch(t, "tenant.*.user.created", "tenant.user.created", false)
}

func TestMatch_Complex_Mixed(t *testing.T) {
	assertMatch(t, "tenant.*.user.>", "tenant.1.user.created", true)
	assertMatch(t, "tenant.*.user.>", "tenant.1.user.profile.updated", true)
	assertMatch(t, "tenant.*.user.>", "tenant.abc.user.settings.avatar.changed", true)
	assertMatch(t, "tenant.*.user.>", "tenant.1.admin.created", false)
	assertMatch(t, "tenant.*.user.>", "tenant.1.user", false)
}

func TestMatch_Complex_LeadingWildcard(t *testing.T) {
	assertMatch(t, "*.created", "user.created", true)
	assertMatch(t, "*.created", "order.created", true)
	assertMatch(t, "*.created", "user.updated", false)
	assertMatch(t, "*.created", "a.b.created", false)
}

func TestMatch_Complex_MultiSegmentMulti(t *testing.T) {
	assertMatch(t, "a.b.>", "a.b.c", true)
	assertMatch(t, "a.b.>", "a.b.c.d.e", true)
	assertMatch(t, "a.b.>", "a.b", false)
	assertMatch(t, "a.b.>", "a.c.d", false)
}

func TestMatch_BackwardCompat(t *testing.T) {
	assertMatch(t, "*", "anything", true)
	assertMatch(t, "*", "a.b.c", true)
	assertMatch(t, "user.*", "user.x", true)
	assertMatch(t, "user.*", "user.x.y", false)
}
