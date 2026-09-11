package log

import (
	"sync"
	"testing"

	"github.com/dangduoc08/ginject/internal/test"
)

func TestWrapLogger_ExactRule_MasksScalarLeaf(t *testing.T) {
	cap := &capturingLogger{}
	l := WrapLogger(cap, []string{"password"})
	l.Info("test", "password", "secret")

	if cap.args()[1] != maskPlaceholder {
		t.Error(test.DiffMessage(cap.args()[1], maskPlaceholder, "exact rule should mask the exact top-level path"))
	}
}

func TestWrapLogger_ExactRule_MatchesByTrailingSegmentAtAnyDepth(t *testing.T) {
	cap := &capturingLogger{}
	l := WrapLogger(cap, []string{"password"})
	l.Info("test", "user", map[string]any{"password": "secret", "name": "duoc"})

	user := cap.args()[1].(map[string]any)
	if user["password"] != maskPlaceholder {
		t.Error(test.DiffMessage(user["password"], maskPlaceholder, "a bare \"password\" rule should mask \"user.password\" too, not just a top-level \"password\" key"))
	}
	if user["name"] != "duoc" {
		t.Error(test.DiffMessage(user["name"], "duoc", "unrelated sibling fields must stay untouched"))
	}
}

func TestWrapLogger_ExactRule_RequiresDotBoundary(t *testing.T) {
	cap := &capturingLogger{}
	l := WrapLogger(cap, []string{"password"})
	l.Info("test", "userpassword", "secret")

	if cap.args()[1] != "secret" {
		t.Error(test.DiffMessage(cap.args()[1], "secret", "\"password\" must not match \"userpassword\" — the match needs a dot boundary, not just a suffix of the raw string"))
	}
}

func TestWrapLogger_MultiSegmentExactRule_MatchesOnlyItsOwnSuffix(t *testing.T) {
	cap := &capturingLogger{}
	l := WrapLogger(cap, []string{"user.password"})
	l.Info("test",
		"user", map[string]any{"password": "secret1"},
		"admin", map[string]any{"password": "secret2"},
	)

	args := cap.args()
	user := args[1].(map[string]any)
	admin := args[3].(map[string]any)
	if user["password"] != maskPlaceholder {
		t.Error(test.DiffMessage(user["password"], maskPlaceholder, "\"user.password\" should mask user.password"))
	}
	if admin["password"] != "secret2" {
		t.Error(test.DiffMessage(admin["password"], "secret2", "\"user.password\" must not match admin.password, a different suffix"))
	}
}

func TestWrapLogger_WildcardRule_MasksNestedLeavesKeepsStructure(t *testing.T) {
	cap := &capturingLogger{}
	l := WrapLogger(cap, []string{"password.*"})
	l.Info("test", "password", map[string]any{
		"person": map[string]any{
			"value": "secret",
		},
	})

	password := cap.args()[1].(map[string]any)
	person, ok := password["person"].(map[string]any)
	if !ok {
		t.Fatalf("expected password.person to remain a map, got %T", password["person"])
	}
	if person["value"] != maskPlaceholder {
		t.Error(test.DiffMessage(person["value"], maskPlaceholder, "password.* should mask the nested leaf value"))
	}
}

func TestWrapLogger_WildcardRule_DoesNotMatchBareTopLevelKey(t *testing.T) {
	cap := &capturingLogger{}
	l := WrapLogger(cap, []string{"password.*"})
	l.Info("test", "password", "secret")

	if cap.args()[1] != "secret" {
		t.Error(test.DiffMessage(cap.args()[1], "secret", "password.* must not match the bare \"password\" path itself, only paths nested under it"))
	}
}

func TestWrapLogger_ExactRule_MasksStructValueWholesale(t *testing.T) {
	type Password struct {
		Value string `log:"value"`
	}
	cap := &capturingLogger{}
	l := WrapLogger(cap, []string{"password"})
	l.Info("test", "password", Password{Value: "secret"})

	if cap.args()[1] != maskPlaceholder {
		t.Error(test.DiffMessage(cap.args()[1], maskPlaceholder, "an exact rule should mask a struct-valued field wholesale without recursing into it"))
	}
}

func TestWrapLogger_TaggedStructUnderWildcardRule(t *testing.T) {
	type Person struct {
		Value string `log:"value"`
	}
	cap := &capturingLogger{}
	l := WrapLogger(cap, []string{"password.*"})
	l.Info("test", "password", map[string]any{"person": Person{Value: "secret"}})

	password := cap.args()[1].(map[string]any)
	person := password["person"].(map[string]any)
	if person["value"] != maskPlaceholder {
		t.Error(test.DiffMessage(person["value"], maskPlaceholder, "wildcard masking should reach into tagged struct fields too, not just plain maps"))
	}
}

func TestWrapLogger_NoRules_PassesValuesThroughUnmasked(t *testing.T) {
	cap := &capturingLogger{}
	l := WrapLogger(cap, nil)
	l.Info("test", "password", "secret")

	if cap.args()[1] != "secret" {
		t.Error(test.DiffMessage(cap.args()[1], "secret", "with no MaskFields configured, nothing should be masked"))
	}
}

func TestWrapLogger_UnrelatedKeyNotMasked(t *testing.T) {
	cap := &capturingLogger{}
	l := WrapLogger(cap, []string{"password"})
	l.Info("test", "username", "duoc")

	if cap.args()[1] != "duoc" {
		t.Error(test.DiffMessage(cap.args()[1], "duoc", "a mask rule for password must not affect an unrelated key"))
	}
}

func TestWrapLogger_MultipleRules(t *testing.T) {
	cap := &capturingLogger{}
	l := WrapLogger(cap, []string{"password", "token.*"})
	l.Info("test",
		"password", "secret1",
		"token", map[string]any{"raw": "secret2"},
		"username", "duoc",
	)

	args := cap.args()
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
	cap := &capturingLogger{}
	l := WrapLogger(cap, []string{"password"})
	l.Info("test", "password", "secret", "dangling")

	args := cap.args()
	if len(args) != 3 {
		t.Fatalf("expected 3 args, got %d", len(args))
	}
	if args[2] != "dangling" {
		t.Error(test.DiffMessage(args[2], "dangling", "a trailing key with no value should be passed through unchanged"))
	}
}

func TestWrapLogger_NonStringKey_PassedThrough(t *testing.T) {
	cap := &capturingLogger{}
	l := WrapLogger(cap, []string{"password"})
	l.Info("test", 42, "secret")

	if cap.args()[1] != "secret" {
		t.Error(test.DiffMessage(cap.args()[1], "secret", "a non-string key means this isn't a well-formed key/value pair; the value must pass through untouched"))
	}
}

func TestWrapLogger_ConcurrentMaskingCallsNoDataRace(t *testing.T) {
	l := WrapLogger(&capturingLogger{}, []string{"password.*"})

	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			l.Info("test", "password", map[string]any{"value": "secret"})
		}()
	}
	wg.Wait()
}

type maskCollectionUser struct {
	Name     string `log:"name"`
	Password string `log:"password"`
}

type maskUntaggedUser struct {
	Name     string
	Password string
}

func TestWrapLogger_SliceOfStructs_MasksEachElement(t *testing.T) {
	cap := &capturingLogger{}
	l := WrapLogger(cap, []string{"password"})
	l.Info("test", "users", []maskCollectionUser{
		{Name: "duoc", Password: "secret-1"},
		{Name: "alice", Password: "secret-2"},
	})

	elems, ok := cap.args()[1].([]any)
	if !ok {
		t.Fatal(test.DiffMessage(cap.args()[1], "[]any", "a slice of structs must be walked, not handed to the sink untouched"))
	}
	if len(elems) != 2 {
		t.Fatalf("expected 2 elements, got %d", len(elems))
	}
	for i, elem := range elems {
		got := elem.(map[string]any)
		if got["password"] != maskPlaceholder {
			t.Error(test.DiffMessage(got["password"], maskPlaceholder, "logging a collection must not bypass masking"))
		}
		if got["name"] == maskPlaceholder {
			t.Errorf("element %d: unrelated fields must stay readable", i)
		}
		_ = i
	}
}

func TestWrapLogger_ArrayOfStructs_MasksEachElement(t *testing.T) {
	cap := &capturingLogger{}
	l := WrapLogger(cap, []string{"password"})
	l.Info("test", "users", [1]maskCollectionUser{{Name: "duoc", Password: "secret"}})

	elems := cap.args()[1].([]any)
	if got := elems[0].(map[string]any); got["password"] != maskPlaceholder {
		t.Error(test.DiffMessage(got["password"], maskPlaceholder, "a fixed-size array must be walked like a slice"))
	}
}

func TestWrapLogger_SliceNestedInMap_MasksEachElement(t *testing.T) {
	cap := &capturingLogger{}
	l := WrapLogger(cap, []string{"password"})
	l.Info("test", "payload", map[string]any{
		"users": []maskCollectionUser{{Name: "duoc", Password: "secret"}},
	})

	payload := cap.args()[1].(map[string]any)
	elems := payload["users"].([]any)
	if got := elems[0].(map[string]any); got["password"] != maskPlaceholder {
		t.Error(test.DiffMessage(got["password"], maskPlaceholder, "a collection reached through a map must still be walked"))
	}
}

func TestWrapLogger_ByteSlice_LeftAsIs(t *testing.T) {
	cap := &capturingLogger{}
	l := WrapLogger(cap, []string{"password"})
	payload := []byte("hello")
	l.Info("test", "body", payload)

	if got, ok := cap.args()[1].([]byte); !ok || string(got) != "hello" {
		t.Error(test.DiffMessage(cap.args()[1], payload, "a byte slice must not be exploded into a list of numbers by the element walk"))
	}
}

func TestWrapLogger_StringSlice_LeftAsIs(t *testing.T) {
	cap := &capturingLogger{}
	l := WrapLogger(cap, []string{"password"})
	l.Info("test", "tags", []string{"a", "b"})

	if _, ok := cap.args()[1].([]string); !ok {
		t.Error(test.DiffMessage(cap.args()[1], []string{"a", "b"}, "a scalar slice needs no walk and must keep its concrete type"))
	}
}

func TestWrapLogger_SliceMatchingRule_MaskedWholesale(t *testing.T) {
	cap := &capturingLogger{}
	l := WrapLogger(cap, []string{"users"})
	l.Info("test", "users", []maskCollectionUser{{Name: "duoc", Password: "secret"}})

	if cap.args()[1] != maskPlaceholder {
		t.Error(test.DiffMessage(cap.args()[1], maskPlaceholder, "a rule naming the collection itself must mask the whole value"))
	}
}

func TestWrapLogger_UntaggedStructField_NeedsMatchingCaseRule(t *testing.T) {
	cap := &capturingLogger{}
	l := WrapLogger(cap, []string{"password"})
	l.Info("test", "users", []maskUntaggedUser{{Name: "duoc", Password: "secret"}})

	got := cap.args()[1].([]any)[0].(map[string]any)
	if got["Password"] != "secret" {
		t.Error(test.DiffMessage(got["Password"], "secret", "rule matching is case-sensitive: a lowercase rule does not reach an untagged Go field name"))
	}

	cap2 := &capturingLogger{}
	l2 := WrapLogger(cap2, []string{"Password"})
	l2.Info("test", "users", []maskUntaggedUser{{Name: "duoc", Password: "secret"}})

	got2 := cap2.args()[1].([]any)[0].(map[string]any)
	if got2["Password"] != maskPlaceholder {
		t.Error(test.DiffMessage(got2["Password"], maskPlaceholder, "a rule matching the exported field name must mask an untagged field"))
	}
}
