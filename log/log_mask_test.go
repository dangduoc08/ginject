package log

import (
	"bytes"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/dangduoc08/ginject/internal/test"
	"github.com/dangduoc08/ginject/pattern"
)

func TestWrapLogger_ExactRule_MasksScalarLeaf(t *testing.T) {
	captured := &capturingLogger{}
	l := WrapLogger(captured, []string{"password"})
	l.Info("test", "password", "secret")

	if captured.args()[1] != maskPlaceholder {
		t.Error(test.DiffMessage(captured.args()[1], maskPlaceholder, "exact rule should mask the exact top-level path"))
	}
}

func TestWrapLogger_ExactRule_MatchesByTrailingSegmentAtAnyDepth(t *testing.T) {
	captured := &capturingLogger{}
	l := WrapLogger(captured, []string{"password"})
	l.Info("test", "user", map[string]any{"password": "secret", "name": "duoc"})

	user := captured.args()[1].(map[string]any)
	if user["password"] != maskPlaceholder {
		t.Error(test.DiffMessage(user["password"], maskPlaceholder, "a bare \"password\" rule should mask \"user.password\" too, not just a top-level \"password\" key"))
	}
	if user["name"] != "duoc" {
		t.Error(test.DiffMessage(user["name"], "duoc", "unrelated sibling fields must stay untouched"))
	}
}

func TestWrapLogger_ExactRule_RequiresDotBoundary(t *testing.T) {
	captured := &capturingLogger{}
	l := WrapLogger(captured, []string{"password"})
	l.Info("test", "userpassword", "secret")

	if captured.args()[1] != "secret" {
		t.Error(test.DiffMessage(captured.args()[1], "secret", "\"password\" must not match \"userpassword\" — the match needs a dot boundary, not just a suffix of the raw string"))
	}
}

func TestWrapLogger_MultiSegmentExactRule_MatchesOnlyItsOwnSuffix(t *testing.T) {
	captured := &capturingLogger{}
	l := WrapLogger(captured, []string{"user.password"})
	l.Info("test",
		"user", map[string]any{"password": "secret1"},
		"admin", map[string]any{"password": "secret2"},
	)

	args := captured.args()
	user := args[1].(map[string]any)
	admin := args[3].(map[string]any)
	if user["password"] != maskPlaceholder {
		t.Error(test.DiffMessage(user["password"], maskPlaceholder, "\"user.password\" should mask user.password"))
	}
	if admin["password"] != "secret2" {
		t.Error(test.DiffMessage(admin["password"], "secret2", "\"user.password\" must not match admin.password, a different suffix"))
	}
}

func TestWrapLogger_WildcardRule_MasksMatchingNodeWhole(t *testing.T) {
	captured := &capturingLogger{}
	l := WrapLogger(captured, []string{"password.**"})
	l.Info("test", "password", map[string]any{
		"person": map[string]any{
			"value": "secret",
		},
	})

	password := captured.args()[1].(map[string]any)
	if password["person"] != maskPlaceholder {
		t.Error(test.DiffMessage(password["person"], maskPlaceholder, "password.** matches password.person, so the whole node is masked"))
	}
}

func TestWrapLogger_WildcardRule_DoesNotMatchBareTopLevelKey(t *testing.T) {
	captured := &capturingLogger{}
	l := WrapLogger(captured, []string{"password.*"})
	l.Info("test", "password", "secret")

	if captured.args()[1] != "secret" {
		t.Error(test.DiffMessage(captured.args()[1], "secret", "password.* must not match the bare \"password\" path itself, only paths nested under it"))
	}
}

func TestWrapLogger_ExactRule_MasksStructValueWholesale(t *testing.T) {
	type Password struct {
		Value string `log:"value"`
	}
	captured := &capturingLogger{}
	l := WrapLogger(captured, []string{"password"})
	l.Info("test", "password", Password{Value: "secret"})

	if captured.args()[1] != maskPlaceholder {
		t.Error(test.DiffMessage(captured.args()[1], maskPlaceholder, "an exact rule should mask a struct-valued field wholesale without recursing into it"))
	}
}

func TestWrapLogger_TaggedStructUnderWildcardRule(t *testing.T) {
	type Person struct {
		Value string `log:"value"`
		Name  string `log:"name"`
	}
	captured := &capturingLogger{}
	l := WrapLogger(captured, []string{"person.value"})
	l.Info("test", "password", map[string]any{"person": Person{Value: "secret", Name: "duoc"}})

	password := captured.args()[1].(map[string]any)
	person := password["person"].(map[string]any)
	if person["value"] != maskPlaceholder || person["name"] != "duoc" {
		t.Error(test.DiffMessage(person, map[string]any{"value": maskPlaceholder, "name": "duoc"}, "masking reaches into tagged struct fields by their tag name"))
	}
}

func TestWrapLogger_NoRules_PassesValuesThroughUnmasked(t *testing.T) {
	captured := &capturingLogger{}
	l := WrapLogger(captured, nil)
	l.Info("test", "password", "secret")

	if captured.args()[1] != "secret" {
		t.Error(test.DiffMessage(captured.args()[1], "secret", "with no MaskFields configured, nothing should be masked"))
	}
}

func TestWrapLogger_UnrelatedKeyNotMasked(t *testing.T) {
	captured := &capturingLogger{}
	l := WrapLogger(captured, []string{"password"})
	l.Info("test", "username", "duoc")

	if captured.args()[1] != "duoc" {
		t.Error(test.DiffMessage(captured.args()[1], "duoc", "a mask rule for password must not affect an unrelated key"))
	}
}

func TestWrapLogger_MultipleRules(t *testing.T) {
	captured := &capturingLogger{}
	l := WrapLogger(captured, []string{"password", "token.*"})
	l.Info("test",
		"password", "secret1",
		"token", map[string]any{"raw": "secret2"},
		"username", "duoc",
	)

	args := captured.args()
	if args[1] != maskPlaceholder {
		t.Error(test.DiffMessage(args[1], maskPlaceholder, "password rule should mask password"))
	}
	token := args[3].(map[string]any)
	if token["raw"] != maskPlaceholder {
		t.Error(test.DiffMessage(token["raw"], maskPlaceholder, "token.* rule should mask token.raw"))
	}
	if args[5] != "duoc" {
		t.Error(test.DiffMessage(args[5], "duoc", "username should be untouched"))
	}
}

func TestWrapLogger_OddArgCount_TrailingKeyPassedThrough(t *testing.T) {
	captured := &capturingLogger{}
	l := WrapLogger(captured, []string{"password"})
	l.Info("test", "password", "secret", "dangling")

	args := captured.args()
	if len(args) != 3 {
		t.Fatal(test.DiffMessage(len(args), 3, "arg count"))
	}
	if args[2] != "dangling" {
		t.Error(test.DiffMessage(args[2], "dangling", "a trailing key with no value should be passed through unchanged"))
	}
}

func TestWrapLogger_NonStringKey_PassedThrough(t *testing.T) {
	captured := &capturingLogger{}
	l := WrapLogger(captured, []string{"password"})
	l.Info("test", 42, "secret")

	if captured.args()[1] != "secret" {
		t.Error(test.DiffMessage(captured.args()[1], "secret", "a non-string key means this isn't a well-formed key/value pair; the value must pass through untouched"))
	}
}

func TestWrapLogger_ConcurrentMaskingCallsNoDataRace(t *testing.T) {
	l := WrapLogger(&capturingLogger{}, []string{"password.*"})

	var wg sync.WaitGroup
	for range 32 {
		wg.Go(func() {
			l.Info("test", "password", map[string]any{"value": "secret"})
		})
	}
	wg.Wait()
}

type maskUser struct {
	Name     string
	Password string
}

func TestWrapLogger_RuleIsCaseInsensitive(t *testing.T) {
	captured := &capturingLogger{}
	WrapLogger(captured, []string{"PassWord"}).Info("test", "user", maskUser{Name: "a", Password: "p"}, "Password", "q")

	user := captured.args()[1].(map[string]any)
	if user["Password"] != maskPlaceholder || captured.args()[3] != maskPlaceholder {
		t.Error(test.DiffMessage(captured.args(), "Password masked in the struct and at the top level", "rules match keys case-insensitively"))
	}
	if user["Name"] != "a" {
		t.Error(test.DiffMessage(user["Name"], "a", "output keys keep their original case"))
	}
}

func TestWrapLogger_SlicesAreTraversed(t *testing.T) {
	captured := &capturingLogger{}
	WrapLogger(captured, []string{"password"}).Info("test",
		"users", []maskUser{{Name: "a", Password: "p1"}},
		"rows", []map[string]any{{"password": "p2"}},
		"tags", []string{"password"},
	)

	users := captured.args()[1].([]any)
	if users[0].(map[string]any)["Password"] != maskPlaceholder {
		t.Error(test.DiffMessage(users, "Password masked", "struct elements of a slice are masked"))
	}
	rows := captured.args()[3].([]any)
	if rows[0].(map[string]any)["password"] != maskPlaceholder {
		t.Error(test.DiffMessage(rows, "password masked", "map elements of a slice are masked"))
	}
	if _, ok := captured.args()[5].([]string); !ok {
		t.Error(test.DiffMessage(captured.args()[5], "[]string unchanged", "a slice of scalars shares its parent's path, so it is passed through"))
	}
}

func TestWrapLogger_WildcardMatchingABranchMasksTheWholeBranch(t *testing.T) {
	captured := &capturingLogger{}
	WrapLogger(captured, []string{"user.*"}).Info("test", "user", map[string]any{"card": map[string]any{"cvv": "1"}, "name": "a"})

	user := captured.args()[1].(map[string]any)
	if user["card"] != maskPlaceholder || user["name"] != maskPlaceholder {
		t.Error(test.DiffMessage(user, "card and name masked whole", "user.* matches every direct child of user"))
	}
}

func TestWrapLogger_WildcardRuleMatchesAnySuffix(t *testing.T) {
	captured := &capturingLogger{}
	WrapLogger(captured, []string{"card.*"}).Info("test", "user", map[string]any{"card": map[string]any{"cvv": "1"}, "name": "a"})

	user := captured.args()[1].(map[string]any)
	card := user["card"].(map[string]any)
	if card["cvv"] != maskPlaceholder || user["name"] != "a" {
		t.Error(test.DiffMessage(user, "card.cvv masked, name kept", "card.* matches user.card.cvv through its suffix card.cvv"))
	}
}

func TestWrapLogger_AttrArgsAreMaskedAndKeepPairsAligned(t *testing.T) {
	captured := &capturingLogger{}
	WrapLogger(captured, []string{"password", "token"}).Info("test",
		slog.String("password", "p"),
		"token", "t",
		slog.Group("http", slog.String("token", "t2"), slog.Int("code", 200)),
		slog.Group("", slog.String("password", "p3")),
		slog.Any("user", map[string]any{"password": "p4"}),
		slog.String("name", "a"),
	)

	args := captured.args()
	checks := []struct {
		got  any
		want any
		desc string
	}{
		{args[0].(slog.Attr).Value.String(), maskPlaceholder, "a leading slog.Attr is masked"},
		{args[2], maskPlaceholder, "the pair after an Attr stays aligned and is masked"},
		{args[3].(slog.Attr).Value.Group()[0].Value.String(), maskPlaceholder, "an Attr inside a group is masked"},
		{args[3].(slog.Attr).Value.Group()[1].Value.Int64(), int64(200), "other Attrs in the group are kept"},
		{args[4].(slog.Attr).Value.Group()[0].Value.String(), maskPlaceholder, "an inline group keeps its parent's path"},
		{args[5].(slog.Attr).Value.Any().(map[string]any)["password"], maskPlaceholder, "an Any Attr is masked inside"},
		{args[6].(slog.Attr).Value.String(), "a", "an unrelated Attr is kept"},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Error(test.DiffMessage(c.got, c.want, c.desc))
		}
	}
}

func TestWrapLogger_NonStringKeyConsumesOneArg(t *testing.T) {
	captured := &capturingLogger{}
	WrapLogger(captured, []string{"password"}).Info("test", 42, "password", "p")

	if captured.args()[2] != maskPlaceholder {
		t.Error(test.DiffMessage(captured.args()[2], maskPlaceholder, "like slog, a non-string key is one !BADKEY arg, so the next pair is still masked"))
	}
}

func TestWrapLogger_UnchangedArgsAreNotCopied(t *testing.T) {
	l := WrapLogger(&capturingLogger{}, []string{"password"}).(*maskingLogger)
	args := []any{"name", "a", "code", 200, "tags", []string{"x"}}
	if out := l.transform(args); &out[0] != &args[0] {
		t.Error(test.DiffMessage("a copy", "the same slice", "transform must not copy args when nothing is masked"))
	}
}

type levelCapturingLogger struct {
	capturingLogger
	minLevel slog.Level
}

func (c *levelCapturingLogger) enabled(level slog.Level) bool { return level >= c.minLevel }

func TestWrapLogger_DisabledLevelSkipsMaskingAndForwarding(t *testing.T) {
	captured := &levelCapturingLogger{minLevel: InfoLevel}
	l := WrapLogger(WrapLogger(captured, []string{"password"}), []string{"token"})
	l.Debug("test", "password", "p")
	if captured.args() != nil {
		t.Error(test.DiffMessage(captured.args(), nil, "a disabled level is dropped before masking, through nested wrappers"))
	}
	l.Info("test", "password", "p", "token", "t")
	if args := captured.args(); args[1] != maskPlaceholder || args[3] != maskPlaceholder {
		t.Error(test.DiffMessage(args, "both masked", "nested wrappers both mask"))
	}
	captured.minLevel = FatalLevel + 1
	l.Fatal("test", "password", "p2")
	if args := captured.args(); args[1] != maskPlaceholder {
		t.Error(test.DiffMessage(args, "forwarded and masked", "Fatal is never gated, so it always reaches os.Exit"))
	}
}

func TestWrapLogger_CyclicValueStopsAtMaxDepth(t *testing.T) {
	selfMap := map[string]any{}
	selfMap["self"] = selfMap
	selfSlice := []any{nil}
	selfSlice[0] = selfSlice

	for name, v := range map[string]any{"map": selfMap, "slice": selfSlice} {
		done := make(chan []any, 1)
		go func() {
			captured := &capturingLogger{}
			WrapLogger(captured, []string{"password"}).Info("test", "v", v)
			done <- captured.args()
		}()
		select {
		case args := <-done:
			if !strings.Contains(fmt.Sprint(args), maxDepthPlaceholder) {
				t.Error(test.DiffMessage(args, "contains "+maxDepthPlaceholder, name))
			}
		case <-time.After(2 * time.Second):
			t.Fatal(test.DiffMessage("still running", "finished", name+": a cyclic value must not hang masking"))
		}
	}
}

func TestWrapLogger_WideValueStopsAtMaxNodesAndStaysMasked(t *testing.T) {
	wide := make([]maskUser, 3000)
	for i := range wide {
		wide[i] = maskUser{Name: "a", Password: "p"}
	}
	captured := &capturingLogger{}
	WrapLogger(captured, []string{"password"}).Info("test", "users", wide)

	users := captured.args()[1].([]any)
	last := users[len(users)-1]
	if s, ok := last.(string); !ok || !strings.HasSuffix(s, " more)") {
		t.Error(test.DiffMessage(last, "… (N more)", "the remaining elements are summarised"))
	}
	for _, u := range users[:len(users)-1] {
		m, ok := u.(map[string]any)
		if _, cut := m["…"]; !ok || cut {
			continue
		}
		if m["Password"] != maskPlaceholder {
			t.Fatal(test.DiffMessage(m, "Password masked", "every element emitted before the limit is masked"))
		}
	}
	if len(users) > maxNodes {
		t.Error(test.DiffMessage(len(users), "at most maxNodes", "the output is bounded"))
	}
}

func TestMatchesSuffix_FastPathEqualsSlowPath(t *testing.T) {
	r := rand.New(rand.NewPCG(3, 4))
	segs := []string{"a", "b", "ab", "*", "**"}
	for range 100000 {
		var rs, ps []string
		for range 1 + r.IntN(3) {
			rs = append(rs, segs[r.IntN(len(segs))])
		}
		for range 1 + r.IntN(5) {
			ps = append(ps, segs[r.IntN(3)])
		}
		rule, path := pattern.NewPattern(strings.Join(rs, ".")), strings.Join(ps, ".")
		if got, want := matchesSuffix(rule, path), matchesAnySuffixReference(rule, path); got != want {
			t.Fatal(test.DiffMessage(got, want, "rule "+rule.Raw()+" path "+path))
		}
	}
}

func matchesAnySuffixReference(rule pattern.Pattern, path string) bool {
	parts := strings.Split(path, ".")
	for i := range parts {
		if rule.Match(strings.Join(parts[i:], ".")) {
			return true
		}
	}
	return false
}

type secretValuer struct{ password string }

func (s secretValuer) LogValue() slog.Value {
	return slog.GroupValue(slog.String("password", s.password), slog.String("name", "a"))
}

func renderAllFormats(args []any) map[string]string {
	outputs := map[string]string{}
	var text, jsonOut, pretty bytes.Buffer
	slog.New(slog.NewTextHandler(&text, nil)).Info("test", args...)
	slog.New(slog.NewJSONHandler(&jsonOut, nil)).Info("test", args...)
	slog.New(newPrettyHandler(&pretty, &prettyHandlerOptions{TimeFormat: time.DateTime})).Info("test", args...)
	outputs["text"], outputs["json"], outputs["pretty"] = text.String(), jsonOut.String(), pretty.String()
	return outputs
}

func TestWrapLogger_LogValuersAreResolvedThenMasked(t *testing.T) {
	v := secretValuer{password: "hunter2"}
	cases := map[string][]any{
		"value":           {"user", v},
		"pointer":         {"user", &v},
		"attr":            {slog.Any("user", v)},
		"group value":     {"user", slog.GroupValue(slog.String("password", "hunter2"), slog.String("name", "a"))},
		"nested in a map": {"wrap", map[string]any{"user": v}},
		"in a slice":      {"users", []any{v}},
	}
	for name, args := range cases {
		captured := &capturingLogger{}
		WrapLogger(captured, []string{"password"}).Info("test", args...)
		for format, out := range renderAllFormats(captured.args()) {
			if strings.Contains(out, "hunter2") || !strings.Contains(out, maskPlaceholder) || !strings.Contains(out, "name") {
				t.Error(test.DiffMessage(out, "password masked, name kept", name+" in "+format))
			}
		}
	}
}

type selfHidingToken struct{ Raw string }

func (selfHidingToken) LogValue() slog.Value { return slog.StringValue("***") }

func TestWrapLogger_NestedLogValuerHidesItselfInEveryFormat(t *testing.T) {
	for _, rules := range [][]string{nil, {"password"}} {
		args := []any{
			"top", selfHidingToken{Raw: "secret-top"},
			"wrap", map[string]any{"t": selfHidingToken{Raw: "secret-map"}},
			"list", []any{selfHidingToken{Raw: "secret-slice"}},
		}
		captured := &capturingLogger{}
		WrapLogger(captured, rules).Info("test", args...)
		for format, out := range renderAllFormats(captured.args()) {
			if strings.Contains(out, "secret-") || strings.Count(out, "***") != 3 {
				t.Error(test.DiffMessage(out, "three *** and no raw value", fmt.Sprint(rules)+" in "+format))
			}
		}
	}
}

func TestWrapLogger_LogValuerChainStopsAtMaxDepth(t *testing.T) {
	var chain any = "end"
	for range 50 {
		chain = map[string]any{"next": wrappedValuer{inner: chain}}
	}
	captured := &capturingLogger{}
	WrapLogger(captured, []string{"password"}).Info("test", "chain", chain)
	if !strings.Contains(fmt.Sprint(captured.args()), maxDepthPlaceholder) {
		t.Error(test.DiffMessage(captured.args(), "contains "+maxDepthPlaceholder, "depth carries through LogValuer hops"))
	}
}

type wrappedValuer struct{ inner any }

func (w wrappedValuer) LogValue() slog.Value { return slog.AnyValue(w.inner) }

type fuzzStruct struct {
	Name string
	Data map[string]any `log:"data"`
}

type fuzzInlineValuer struct{ key, secret string }

func (v fuzzInlineValuer) LogValue() slog.Value {
	return slog.GroupValue(slog.Any("", slog.GroupValue(slog.String(v.key, v.secret))))
}

type fuzzValuer struct{ key, secret string }

func (v fuzzValuer) LogValue() slog.Value { return slog.GroupValue(slog.String(v.key, v.secret)) }

func alphanumeric(s string, lower bool) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z' && !lower:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func FuzzWrapLogger_MaskedSecretNeverReachesOutput(f *testing.F) {
	f.Add("password", "hunter2", uint8(0), uint8(0), uint8(0))
	f.Add("token", "abc", uint8(1), uint8(2), uint8(3))
	f.Add("cvv", "123", uint8(4), uint8(5), uint8(7))
	f.Fuzz(func(t *testing.T, rawKey, rawSecret string, shape, nesting, mode uint8) {
		key := alphanumeric(rawKey, true)
		if key == "" {
			return
		}
		secret := "S3CR-" + alphanumeric(rawSecret, false) + "-T"
		dataKey := key
		if mode&1 != 0 {
			dataKey = strings.ToUpper(key)
		}

		var value any
		switch shape % 9 {
		case 0:
			value = map[string]any{dataKey: secret}
		case 1:
			value = []any{map[string]any{dataKey: secret}}
		case 2:
			value = fuzzValuer{key: dataKey, secret: secret}
		case 3:
			value = slog.GroupValue(slog.String(dataKey, secret))
		case 4:
			value = map[string]string{dataKey: secret}
		case 5:
			value = []fuzzStruct{{Name: "a", Data: map[string]any{dataKey: secret}}}
		case 6:
			value = fuzzInlineValuer{key: dataKey, secret: secret}
		case 7:
			value = []slog.Attr{slog.String(dataKey, secret)}
		default:
			value = slog.AnyValue(slog.String(dataKey, secret))
		}
		for range nesting % 6 {
			value = map[string]any{"n": value}
		}

		var args []any
		if mode&2 != 0 {
			args = append(args, 42)
		}
		if mode&4 != 0 {
			args = append(args, slog.Any("root", value))
		} else {
			args = append(args, "root", value)
		}

		captured := &capturingLogger{}
		WrapLogger(captured, []string{key}).Info("test", args...)
		for format, out := range renderAllFormats(captured.args()) {
			if strings.Contains(out, secret) {
				t.Fatal(test.DiffMessage(out, "no "+secret, format))
			}
		}
	})
}

func FuzzMatchesSuffix_FastPathEqualsSlowPath(f *testing.F) {
	f.Add("password", "user.password")
	f.Add("card.*", "user.card.cvv")
	f.Add("a.**", "x.a.b.c")
	f.Fuzz(func(t *testing.T, raw, path string) {
		rule := pattern.NewPattern(raw)
		if got, want := matchesSuffix(rule, path), matchesAnySuffixReference(rule, path); got != want {
			t.Fatal(test.DiffMessage(got, want, "rule "+raw+" path "+path))
		}
	})
}

type inlineSelf struct{}

func (inlineSelf) LogValue() slog.Value { return slog.GroupValue(slog.Any("", inlineSelf{})) }

func TestWrapLogger_InlineGroupCycleStops(t *testing.T) {
	for name, v := range map[string]any{
		"map":    map[string]any{"v": inlineSelf{}},
		"slice":  []any{inlineSelf{}},
		"struct": struct{ V any }{V: inlineSelf{}},
	} {
		done := make(chan []any, 1)
		go func() {
			captured := &capturingLogger{}
			WrapLogger(captured, nil).Info("test", "root", v)
			done <- captured.args()
		}()
		select {
		case args := <-done:
			if !strings.Contains(fmt.Sprint(args), maxDepthPlaceholder) {
				t.Error(test.DiffMessage(args, "contains "+maxDepthPlaceholder, name))
			}
		case <-time.After(2 * time.Second):
			t.Fatal(test.DiffMessage("still running", "finished", name+": an inline group cycle must stop"))
		}
	}
}

func TestWrapLogger_NestedAttrsAreMaskedLikeGroups(t *testing.T) {
	cases := map[string][]any{
		"attrs as a value":                  {"attrs", []slog.Attr{slog.String("password", "hunter2"), slog.String("name", "a")}},
		"attrs in a map":                    {"wrap", map[string]any{"attrs": []slog.Attr{slog.String("password", "hunter2"), slog.String("name", "a")}}},
		"attr as a value":                   {"attr", slog.Group("g", slog.String("password", "hunter2"), slog.String("name", "a"))},
		"attr in a slice":                   {"list", []any{slog.Group("g", slog.String("password", "hunter2"), slog.String("name", "a"))}},
		"value holding an attr in a map":    {"wrap", map[string]any{"v": slog.AnyValue(slog.Group("g", slog.String("password", "hunter2"), slog.String("name", "a")))}},
		"attrs holding an attr in a struct": {"wrap", struct{ Attrs []slog.Attr }{Attrs: []slog.Attr{slog.Any("inner", slog.Group("g", slog.String("password", "hunter2"), slog.String("name", "a")))}}},
	}
	for name, args := range cases {
		captured := &capturingLogger{}
		WrapLogger(captured, []string{"password"}).Info("test", args...)
		for format, out := range renderAllFormats(captured.args()) {
			if strings.Contains(out, "hunter2") || !strings.Contains(out, maskPlaceholder) || !strings.Contains(out, "name") {
				t.Error(test.DiffMessage(out, "password masked, name kept", name+" in "+format))
			}
		}
	}
}
