package throttler

import (
	"testing"
	"time"

	"github.com/dangduoc08/ginject/ctx"
	"github.com/dangduoc08/ginject/exception"
	"github.com/dangduoc08/ginject/internal/test"
)

func wsCtxFor(connID, topic, operation string) *ctx.WSContext {
	c := ctx.NewWSContext()
	c.SetID()
	c.SetEvent(connID, "chat.*", topic, operation)
	return c
}

func TestWSThrottler_AllowsUpToTheLimitThenRejects(t *testing.T) {
	guard := WSThrottler{Limit: 3, TTL: time.Minute}.NewGuard()

	for i := 0; i < 3; i++ {
		if !guard.CanActivate(wsCtxFor("conn-1", "chat.1", "publish")) {
			t.Fatalf("request %d within the limit must be allowed", i+1)
		}
	}

	defer func() {
		rec := recover()
		if rec == nil {
			t.Fatal(test.DiffMessage(nil, "panic", "a WebSocket connection past its publish budget must be rejected; without this a single socket can flood the broker"))
		}
		ex, ok := rec.(exception.Exception)
		if !ok {
			t.Fatalf("expected an Exception, got %T", rec)
		}
		if ex.GetCode() != 1008 {
			t.Error(test.DiffMessage(ex.GetCode(), 1008, "a rate-limit rejection must use the WebSocket policy-violation code"))
		}
	}()

	guard.CanActivate(wsCtxFor("conn-1", "chat.1", "publish"))
}

func TestWSThrottler_BudgetsAreScopedPerConnection(t *testing.T) {
	guard := WSThrottler{Limit: 1, TTL: time.Minute}.NewGuard()

	if !guard.CanActivate(wsCtxFor("conn-a", "chat.1", "publish")) {
		t.Fatal("first request for conn-a must be allowed")
	}
	if !guard.CanActivate(wsCtxFor("conn-b", "chat.1", "publish")) {
		t.Error(test.DiffMessage(false, true, "one noisy connection must not consume another connection's budget"))
	}
}

func TestWSThrottler_TopicScopedKeyIsolatesTopics(t *testing.T) {
	guard := WSThrottler{Limit: 1, TTL: time.Minute, KeyFunc: ConnTopicScopedKeyFunc}.NewGuard()

	if !guard.CanActivate(wsCtxFor("conn-a", "chat.1", "publish")) {
		t.Fatal("first publish to chat.1 must be allowed")
	}
	if !guard.CanActivate(wsCtxFor("conn-a", "chat.2", "publish")) {
		t.Error(test.DiffMessage(false, true, "a per-topic budget must not be shared across topics"))
	}
}

func TestWSThrottler_DefaultsAreSafe(t *testing.T) {
	guard := WSThrottler{}.NewGuard()

	if guard.Limit <= 0 || guard.TTL <= 0 || guard.KeyFunc == nil || guard.Backend == nil {
		t.Error(test.DiffMessage(guard, "fully defaulted", "a zero-value WSThrottler must still be usable"))
	}
}
