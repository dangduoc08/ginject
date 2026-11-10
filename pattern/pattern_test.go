package pattern_test

import (
	"flag"
	"fmt"
	"math/rand/v2"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/dangduoc08/ginject/internal/test"
	"github.com/dangduoc08/ginject/pattern"
)

func TestNewPattern_Kind(t *testing.T) {
	cases := []struct {
		raw  string
		want pattern.Kind
	}{
		{"user.created", pattern.KindExact},
		{"a", pattern.KindExact},
		{"", pattern.KindExact},
		{"a*b", pattern.KindExact},
		{"a.**.b", pattern.KindExact},
		{"**", pattern.KindGlobal},
		{"user.**", pattern.KindSuffixWildcard},
		{"a.b.c.**", pattern.KindSuffixWildcard},
		{"*", pattern.KindComplex},
		{"user.*", pattern.KindComplex},
		{"chat.*.msg", pattern.KindComplex},
		{"tenant.*.user.**", pattern.KindComplex},
	}
	for _, c := range cases {
		if got := pattern.NewPattern(c.raw).Kind(); got != c.want {
			t.Error(test.DiffMessage(got, c.want, "Kind of "+c.raw))
		}
	}
}

func TestNewPattern_SimplePrefix(t *testing.T) {
	cases := map[string]string{"user.**": "user", "a.b.**": "a.b", "a.b.c.**": "a.b.c"}
	for raw, want := range cases {
		if got := pattern.NewPattern(raw).SimplePrefix(); got != want {
			t.Error(test.DiffMessage(got, want, "SimplePrefix of "+raw))
		}
	}
}

func TestPattern_RawAndIsExact(t *testing.T) {
	cases := []struct {
		raw     string
		isExact bool
	}{
		{"user.password", true},
		{"user.*", false},
		{"user.**", false},
		{"**", false},
	}
	for _, c := range cases {
		p := pattern.NewPattern(c.raw)
		if p.Raw() != c.raw || p.IsExact() != c.isExact {
			t.Error(test.DiffMessage([]any{p.Raw(), p.IsExact()}, []any{c.raw, c.isExact}, "Raw/IsExact of "+c.raw))
		}
	}
}

func TestMatch_SemanticsTable(t *testing.T) {
	cases := []struct {
		pattern string
		topic   string
		want    bool
	}{
		{"chat.room1", "chat.room1", true},
		{"chat.room1", "chat.room2", false},

		{"chat.*", "chat.room1", true},
		{"chat.*", "chat.room1.typing", false},
		{"chat.*", "chat", false},
		{"chat.*.typing", "chat.room1.typing", true},
		{"chat.*.typing", "chat.room1.thread5.typing", false},
		{"*.room1", "chat.room1", true},
		{"*", "chat", true},
		{"*", "chat.room1", false},

		{"chat.**", "chat.room1", true},
		{"chat.**", "chat.room1.typing", true},
		{"chat.**", "chat", false},
		{"chat.**", "chatx.room1", false},
		{"chat.*.**", "chat.room1", false},
		{"chat.*.**", "chat.room1.typing", true},
		{"**", "chat", true},
		{"**", "chat.room1.typing", true},

		{"a*b", "axxb", false},
		{"a*b", "a*b", true},
		{"a.**.b", "a.x.b", false},
		{"a.**.b", "a.**.b", true},
		{"", "", true},

		{"chat.**", "chat.", false},
		{"chat.*.**", "chat.room1.", true},
	}
	for _, c := range cases {
		assertMatch(t, c.pattern, c.topic, c.want)
	}
}

func TestMatch_ComplexEqualsSplitReference(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 1))
	patternSegs := []string{"a", "b", "*", "**", ""}
	topicSegs := []string{"a", "b", "c", ""}
	for range 300000 {
		var ps, ts []string
		for range 1 + r.IntN(5) {
			ps = append(ps, patternSegs[r.IntN(len(patternSegs))])
		}
		for range 1 + r.IntN(6) {
			ts = append(ts, topicSegs[r.IntN(len(topicSegs))])
		}
		raw, topic := strings.Join(ps, "."), strings.Join(ts, ".")
		if pattern.NewPattern(raw).Kind() != pattern.KindComplex {
			continue
		}
		if got, want := pattern.NewPattern(raw).Match(topic), splitReferenceMatch(raw, topic); got != want {
			t.Fatal(test.DiffMessage(got, want, "pattern "+raw+" topic "+topic))
		}
	}
}

func TestMatch_LongHostileTopicIsCheap(t *testing.T) {
	p := pattern.NewPattern("a.*.z")
	topic := strings.Repeat("seg.", 250000) + "end"
	start := time.Now()
	p.Match(topic)
	if elapsed := time.Since(start); elapsed > time.Millisecond {
		t.Error(test.DiffMessage(elapsed.String(), "< 1ms", "matching stops at the first mismatching segment"))
	}
}

var updateMatching = flag.Bool("update-matching", false, "rewrite MATCHING.md from the current Match behaviour")

type matchingCase struct {
	topic string
	note  string
}

type matchingScenario struct {
	title   string
	desc    string
	pattern string
	cases   []matchingCase
}

var matchingScenarios = []matchingScenario{
	{"1. Exact", "No wildcard: the topic must be identical.", "chat.room1",
		[]matchingCase{{"chat.room1", ""}, {"chat.room2", ""}, {"chat.room1.typing", "longer topics don't match"}, {"Chat.room1", "case-sensitive"}}},
	{"2. `*` matches exactly one segment", "`*` stands for one segment in any position, including the last.", "chat.*",
		[]matchingCase{{"chat.room1", ""}, {"chat.room1.typing", "two segments after `chat`"}, {"chat", "no segment after `chat`"}}},
	{"3. `*` in the middle", "", "chat.*.typing",
		[]matchingCase{{"chat.room1.typing", ""}, {"chat.room1.thread5.typing", "`*` doesn't stretch over two segments"}, {"chat.typing", "no segment for `*`"}}},
	{"4. `*` first", "", "*.room1",
		[]matchingCase{{"chat.room1", ""}, {"game.room1", ""}, {"room1", ""}}},
	{"5. `*` alone", "Matches single-segment topics only.", "*",
		[]matchingCase{{"chat", ""}, {"chat.room1", "use `**` to match everything"}}},
	{"6. `**` matches one or more trailing segments", "`**` is only special as the last segment.", "chat.**",
		[]matchingCase{{"chat.room1", ""}, {"chat.room1.typing", ""}, {"chat.room1.thread5.typing", ""}, {"chat", "`**` needs at least one segment"}, {"chatx.room1", "the prefix must match a whole segment"}}},
	{"7. `*` then `**`", "", "chat.*.**",
		[]matchingCase{{"chat.room1", "`**` needs one more segment"}, {"chat.room1.typing", ""}, {"chat.room1.thread5.typing", ""}}},
	{"8. `**` alone", "Matches every topic.", "**",
		[]matchingCase{{"chat", ""}, {"chat.room1.typing", ""}}},
	{"9. Literals", "`*` inside a segment, and `**` anywhere but last, are plain characters.", "a.**.b",
		[]matchingCase{{"a.x.b", "`**` in the middle is not a wildcard"}, {"a.**.b", "only the literal text matches"}}},
	{"10. Masking keys (`MaskFields`)", "The same syntax selects log attribute paths to redact.", "user.*",
		[]matchingCase{{"user.password", "one level under `user`"}, {"user.card.cvv", "use `user.**` for all levels"}}},
}

func TestMatchingTableIsUpToDate(t *testing.T) {
	want := renderMatchingTable()
	if *updateMatching {
		if err := os.WriteFile("MATCHING.md", []byte(want), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	got, err := os.ReadFile("MATCHING.md")
	if err != nil {
		t.Fatal(test.DiffMessage(err.Error(), "MATCHING.md present", "run with -update-matching to create it"))
	}
	if string(got) != want {
		t.Error(test.DiffMessage("MATCHING.md is stale", "MATCHING.md matches Match", "run go test ./pattern -run TestMatchingTableIsUpToDate -update-matching"))
	}
}

func renderMatchingTable() string {
	var b strings.Builder
	b.WriteString("# Pattern matching in `pattern` (Pattern.Match)\n\n")
	b.WriteString("Every row below is the actual output of `Pattern.Match`, generated by `TestMatchingTableIsUpToDate` in `pattern_test.go`. The test fails if this file drifts from the code. To regenerate it, run `go test ./pattern -run TestMatchingTableIsUpToDate -update-matching`.\n\n")
	b.WriteString("Patterns and topics are dot-separated segments. The syntax is glob-style (D-030): `*` is exactly one segment, and a trailing `**` is one or more. The broker's topics (R-2) and the logger's `MaskFields` use the same syntax.\n\n")
	for _, sc := range matchingScenarios {
		p := pattern.NewPattern(sc.pattern)
		b.WriteString("## " + sc.title + "\n\n")
		if sc.desc != "" {
			b.WriteString(sc.desc + "\n\n")
		}
		b.WriteString("Pattern: `" + sc.pattern + "`\n\n")
		b.WriteString("| Topic | Match | Note |\n|---|---|---|\n")
		for _, c := range sc.cases {
			result := "❌ no"
			if p.Match(c.topic) {
				result = "✅ yes"
			}
			note := c.note
			if note == "" {
				note = "—"
			}
			fmt.Fprintf(&b, "| `%s` | %s | %s |\n", c.topic, result, note)
		}
		b.WriteString("\n")
	}
	return b.String()
}

func splitReferenceMatch(raw, topic string) bool {
	segs, parts := strings.Split(raw, "."), strings.Split(topic, ".")
	for i, s := range segs {
		if s == "**" && i == len(segs)-1 {
			return i < len(parts)
		}
		if i >= len(parts) || (s != "*" && s != parts[i]) {
			return false
		}
	}
	return len(segs) == len(parts)
}

func assertMatch(t *testing.T, patternStr, topic string, want bool) {
	t.Helper()
	if got := pattern.NewPattern(patternStr).Match(topic); got != want {
		t.Error(test.DiffMessage(got, want, "Match("+patternStr+", "+topic+")"))
	}
}
