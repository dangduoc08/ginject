package accesslog

import (
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/dangduoc08/ginject/event"
	"github.com/dangduoc08/ginject/internal/test"
	"github.com/dangduoc08/ginject/log"
	"github.com/dangduoc08/ginject/trace"
)

func stageEqual(a, b []map[string]log.Colored) bool {
	return slices.EqualFunc(a, b, func(x, y map[string]log.Colored) bool {
		return maps.Equal(x, y)
	})
}

type recordedLog struct {
	msg  string
	args []any
}

type fakeLogger struct {
	ch chan recordedLog
}

func (f *fakeLogger) Debug(msg string, args ...any) {}
func (f *fakeLogger) Info(msg string, args ...any)  { f.ch <- recordedLog{msg: msg, args: args} }
func (f *fakeLogger) Warn(msg string, args ...any)  {}
func (f *fakeLogger) Error(msg string, args ...any) {}
func (f *fakeLogger) Fatal(msg string, args ...any) {}

func recvLog(t *testing.T, ch chan recordedLog) recordedLog {
	t.Helper()
	select {
	case l := <-ch:
		return l
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for a logged entry")
		return recordedLog{}
	}
}

func TestAccessLog_FlushesAllStagesInOneLogCallOnDone(t *testing.T) {
	logger := &fakeLogger{ch: make(chan recordedLog, 10)}
	ev := event.NewEvent()
	al := NewAccessLog(&AccessLogConfig{Event: ev, Logger: logger})

	ev.Emit(trace.EventName, trace.Event{ID: "req-1", Stage: trace.StageMiddleware, Name: "M1"})
	ev.Emit(trace.EventName, trace.Event{ID: "req-1", Stage: trace.StageGuard, Name: "G1"})
	ev.Emit(trace.EventName, trace.Event{ID: "req-1", Stage: trace.StageComplete, Code: 200, Target: "Request"})

	got := recvLog(t, logger.ch)
	if got.msg != "Request" {
		t.Error(test.DiffMessage(got.msg, "Request", "the whole request should be logged in a single call"))
	}
	if got.args[5] != 200 {
		t.Error(test.DiffMessage(got.args[5], 200, "the log call should carry the final status code"))
	}
	wantMiddleware := []map[string]log.Colored{{"M1": {Text: "0s (0%)", Color: colorForDuration(0)}}}
	wantGuard := []map[string]log.Colored{{"G1": {Text: "0s (0%)", Color: colorForDuration(0)}}}
	if got.args[8] != "middleware" || !stageEqual(got.args[9].([]map[string]log.Colored), wantMiddleware) {
		t.Error(test.DiffMessage([]any{got.args[8], got.args[9]}, []any{"middleware", wantMiddleware}, "the middleware stage should be logged as its own key/ordered-name-keyed-duration-list pair"))
	}
	if got.args[10] != "guard" || !stageEqual(got.args[11].([]map[string]log.Colored), wantGuard) {
		t.Error(test.DiffMessage([]any{got.args[10], got.args[11]}, []any{"guard", wantGuard}, "the guard stage should be logged as its own key/ordered-name-keyed-duration-list pair"))
	}

	select {
	case l := <-logger.ch:
		t.Errorf("expected exactly one log call, got a second one: %+v", l)
	default:
	}

	al.mu.Lock()
	_, exists := al.buf["req-1"]
	al.mu.Unlock()
	if exists {
		t.Error(test.DiffMessage(true, false, "the buffer entry for a flushed request ID should be deleted"))
	}
}

func TestAccessLog_SeparatesEntriesByID(t *testing.T) {
	logger := &fakeLogger{ch: make(chan recordedLog, 10)}
	ev := event.NewEvent()
	al := NewAccessLog(&AccessLogConfig{Event: ev, Logger: logger})

	ev.Emit(trace.EventName, trace.Event{ID: "req-a", Stage: trace.StageMiddleware, Name: "A1"})
	ev.Emit(trace.EventName, trace.Event{ID: "req-b", Stage: trace.StageMiddleware, Name: "B1"})
	ev.Emit(trace.EventName, trace.Event{ID: "req-a", Stage: trace.StageComplete, Code: 200, Target: "Request"})

	got := recvLog(t, logger.ch)
	wantMiddleware := []map[string]log.Colored{{"A1": {Text: "0s (0%)", Color: colorForDuration(0)}}}
	if got.args[8] != "middleware" || !stageEqual(got.args[9].([]map[string]log.Colored), wantMiddleware) {
		t.Error(test.DiffMessage([]any{got.args[8], got.args[9]}, []any{"middleware", wantMiddleware}, "req-a's log call should only carry req-a's own entries"))
	}

	al.mu.Lock()
	entries, exists := al.buf["req-b"]
	al.mu.Unlock()
	if !exists || len(entries) != 1 || entries[0].Name != "B1" {
		t.Error(test.DiffMessage(entries, "[B1]", "req-b's entry should remain buffered, untouched by req-a's flush"))
	}
}

func TestAccessLog_StripsPackagePrefixFromName(t *testing.T) {
	logger := &fakeLogger{ch: make(chan recordedLog, 10)}
	ev := event.NewEvent()
	NewAccessLog(&AccessLogConfig{Event: ev, Logger: logger})

	ev.Emit(trace.EventName, trace.Event{ID: "req-1", Stage: trace.StageMiddleware, Name: "cors.CORS"})
	ev.Emit(trace.EventName, trace.Event{ID: "req-1", Stage: trace.StageComplete, Code: 200, Target: "Request"})

	got := recvLog(t, logger.ch)
	want := []map[string]log.Colored{{"CORS": {Text: "0s (0%)", Color: colorForDuration(0)}}}
	if !stageEqual(got.args[9].([]map[string]log.Colored), want) {
		t.Error(test.DiffMessage(got.args[9], want, "the package prefix should be stripped from the logged name"))
	}
}

func TestAccessLog_DurationTextIncludesPercentOfTotal(t *testing.T) {
	logger := &fakeLogger{ch: make(chan recordedLog, 10)}
	ev := event.NewEvent()
	NewAccessLog(&AccessLogConfig{Event: ev, Logger: logger})

	ev.Emit(trace.EventName, trace.Event{ID: "req-1", Stage: trace.StageMiddleware, Name: "M1", Duration: 10 * time.Millisecond})
	ev.Emit(trace.EventName, trace.Event{ID: "req-1", Stage: trace.StageHandler, Name: "H1", Duration: 90 * time.Millisecond})
	ev.Emit(trace.EventName, trace.Event{ID: "req-1", Stage: trace.StageComplete, Code: 200, Target: "Request"})

	got := recvLog(t, logger.ch)
	want := []map[string]log.Colored{{"M1": {Text: "10ms (10%)", Color: colorForDuration(10 * time.Millisecond)}}}
	if !stageEqual(got.args[9].([]map[string]log.Colored), want) {
		t.Error(test.DiffMessage(got.args[9], want, "a stage entry's duration text should include its percent share of the total request duration"))
	}
}

func TestAccessLog_TotalEqualsSumOfStageDurations(t *testing.T) {
	logger := &fakeLogger{ch: make(chan recordedLog, 10)}
	ev := event.NewEvent()
	NewAccessLog(&AccessLogConfig{Event: ev, Logger: logger})

	ev.Emit(trace.EventName, trace.Event{ID: "req-1", Stage: trace.StageMiddleware, Name: "M1", Duration: 10 * time.Millisecond})
	ev.Emit(trace.EventName, trace.Event{ID: "req-1", Stage: trace.StageGuard, Name: "G1", Duration: 20 * time.Millisecond})
	ev.Emit(trace.EventName, trace.Event{ID: "req-1", Stage: trace.StageHandler, Name: "H1", Duration: 70 * time.Millisecond})
	ev.Emit(trace.EventName, trace.Event{ID: "req-1", Stage: trace.StageComplete, Code: 200, Target: "Request", Duration: 999 * time.Millisecond})

	got := recvLog(t, logger.ch)
	wantTotal := formatDuration(100 * time.Millisecond)
	if got.args[len(got.args)-1] != wantTotal {
		t.Error(test.DiffMessage(got.args[len(got.args)-1], wantTotal, "logged total must equal the sum of all stage durations, ignoring the wall-clock StageComplete duration"))
	}
}

func extractPercentFromColoredText(t *testing.T, text string) int {
	t.Helper()
	open := strings.IndexByte(text, '(')
	closeIdx := strings.IndexByte(text, '%')
	p, err := strconv.Atoi(text[open+1 : closeIdx])
	if err != nil {
		t.Fatalf("failed to parse percent from %q: %v", text, err)
	}
	return p
}

func sumLoggedPercents(t *testing.T, args []any) int {
	t.Helper()
	sum := 0
	for i := 8; i < len(args)-2; i += 2 {
		entries, ok := args[i+1].([]map[string]log.Colored)
		if !ok {
			continue
		}
		for _, entry := range entries {
			for _, colored := range entry {
				sum += extractPercentFromColoredText(t, colored.Text)
			}
		}
	}
	return sum
}

func TestAccessLog_PercentagesSumToExactlyOneHundred(t *testing.T) {
	logger := &fakeLogger{ch: make(chan recordedLog, 10)}
	ev := event.NewEvent()
	NewAccessLog(&AccessLogConfig{Event: ev, Logger: logger})

	// 1ms/1ms/1ms of a 3ms total is 33.33% each; naive independent rounding
	// would floor every entry to 33% and sum to 99%.
	ev.Emit(trace.EventName, trace.Event{ID: "req-1", Stage: trace.StageMiddleware, Name: "M1", Duration: 1 * time.Millisecond})
	ev.Emit(trace.EventName, trace.Event{ID: "req-1", Stage: trace.StageGuard, Name: "G1", Duration: 1 * time.Millisecond})
	ev.Emit(trace.EventName, trace.Event{ID: "req-1", Stage: trace.StageHandler, Name: "H1", Duration: 1 * time.Millisecond})
	ev.Emit(trace.EventName, trace.Event{ID: "req-1", Stage: trace.StageComplete, Code: 200, Target: "Request"})

	got := recvLog(t, logger.ch)

	sum := sumLoggedPercents(t, got.args)
	if sum != 100 {
		t.Error(test.DiffMessage(sum, 100, "stage percentages must sum to exactly 100%"))
	}
}

func TestAccessLog_PreservesExecutionOrderNotAlphabetical(t *testing.T) {
	logger := &fakeLogger{ch: make(chan recordedLog, 10)}
	ev := event.NewEvent()
	NewAccessLog(&AccessLogConfig{Event: ev, Logger: logger})

	ev.Emit(trace.EventName, trace.Event{ID: "req-1", Stage: trace.StagePreInterceptor, Name: "C"})
	ev.Emit(trace.EventName, trace.Event{ID: "req-1", Stage: trace.StagePreInterceptor, Name: "B"})
	ev.Emit(trace.EventName, trace.Event{ID: "req-1", Stage: trace.StagePreInterceptor, Name: "A"})
	ev.Emit(trace.EventName, trace.Event{ID: "req-1", Stage: trace.StagePostInterceptor, Name: "A"})
	ev.Emit(trace.EventName, trace.Event{ID: "req-1", Stage: trace.StagePostInterceptor, Name: "B"})
	ev.Emit(trace.EventName, trace.Event{ID: "req-1", Stage: trace.StagePostInterceptor, Name: "C"})
	ev.Emit(trace.EventName, trace.Event{ID: "req-1", Stage: trace.StageComplete, Code: 200, Target: "Request"})

	got := recvLog(t, logger.ch)

	var pre, post []map[string]log.Colored
	for i := 8; i < len(got.args)-2; i += 2 {
		switch got.args[i] {
		case trace.StagePreInterceptor:
			pre = got.args[i+1].([]map[string]log.Colored)
		case trace.StagePostInterceptor:
			post = got.args[i+1].([]map[string]log.Colored)
		}
	}

	entryNames := func(entries []map[string]log.Colored) []string {
		names := make([]string, len(entries))
		for i, entry := range entries {
			for name := range entry {
				names[i] = name
			}
		}
		return names
	}

	wantPreNames := []string{"C", "B", "A"}
	if !slices.Equal(entryNames(pre), wantPreNames) {
		t.Error(test.DiffMessage(entryNames(pre), wantPreNames, "pre-interceptor entries must be logged in registration/emission order, not sorted alphabetically"))
	}

	wantPostNames := []string{"A", "B", "C"}
	if !slices.Equal(entryNames(post), wantPostNames) {
		t.Error(test.DiffMessage(entryNames(post), wantPostNames, "post-interceptor entries must be logged in emission order, matching FILO relative to pre-interceptor"))
	}
}

func TestColorForDuration_Thresholds(t *testing.T) {
	cases := []struct {
		d    time.Duration
		want log.Color
	}{
		{5 * time.Millisecond, log.ColorGray},
		{9999 * time.Microsecond, log.ColorGray},
		{10 * time.Millisecond, log.ColorGreen},
		{49 * time.Millisecond, log.ColorGreen},
		{50 * time.Millisecond, log.ColorBlue},
		{99 * time.Millisecond, log.ColorBlue},
		{100 * time.Millisecond, log.ColorYellow},
		{299 * time.Millisecond, log.ColorYellow},
		{300 * time.Millisecond, log.ColorOrange},
		{999 * time.Millisecond, log.ColorOrange},
		{1000 * time.Millisecond, log.ColorRed},
		{2 * time.Second, log.ColorRed},
	}
	for _, c := range cases {
		if got := colorForDuration(c.d); got != c.want {
			t.Error(test.DiffMessage(got, c.want, "colorForDuration("+c.d.String()+")"))
		}
	}
}

func TestFormatDuration_RoundsToTwoDecimals(t *testing.T) {
	cases := []struct {
		d    time.Duration
		want string
	}{
		{1234 * time.Nanosecond, "1.23µs"},
		{1234567 * time.Nanosecond, "1.23ms"},
		{1234567891 * time.Nanosecond, "1.23s"},
		{500 * time.Nanosecond, "500ns"},
	}
	for _, c := range cases {
		if got := formatDuration(c.d); got != c.want {
			t.Error(test.DiffMessage(got, c.want, "formatDuration("+c.d.String()+")"))
		}
	}
}

func TestShortName(t *testing.T) {
	cases := []struct {
		name string
		want string
	}{
		{"cors.CORS", "CORS"},
		{"pkg.sub.Type", "Type"},
		{"NoPackage", "NoPackage"},
		{"", ""},
		{"trailing.", ""},
	}
	for _, c := range cases {
		if got := shortName(c.name); got != c.want {
			t.Error(test.DiffMessage(got, c.want, "shortName("+c.name+")"))
		}
	}
}

func sumInts(vals []int) int {
	sum := 0
	for _, v := range vals {
		sum += v
	}
	return sum
}

func TestDistributePercentages_SumsToExactlyOneHundred(t *testing.T) {
	entries := []trace.Event{
		{Duration: 1 * time.Millisecond},
		{Duration: 1 * time.Millisecond},
		{Duration: 1 * time.Millisecond},
	}
	total := sumDurations(entries)
	got := distributePercentages(entries, total)
	if sum := sumInts(got); sum != 100 {
		t.Error(test.DiffMessage(sum, 100, "three equal thirds must still sum to exactly 100"))
	}
}

func TestDistributePercentages_LargestRemainderGetsPriority(t *testing.T) {
	entries := []trace.Event{
		{Duration: 50},
		{Duration: 30},
		{Duration: 21},
	}
	total := time.Duration(101)

	// exact shares: 49.5049%, 29.7029%, 20.7921%
	// floor shares: 49, 29, 20 (sum 98, needs +2)
	// remainders:   .5049, .7029, .7921 -> entry[2] then entry[1] get the +1
	want := []int{49, 30, 21}
	got := distributePercentages(entries, total)
	if !slices.Equal(got, want) {
		t.Error(test.DiffMessage(got, want, "entries with the largest fractional remainder must receive the extra point first"))
	}
	if sum := sumInts(got); sum != 100 {
		t.Error(test.DiffMessage(sum, 100, "percentages must sum to exactly 100"))
	}
}

func TestDistributePercentages_TiesBrokenByOriginalOrder(t *testing.T) {
	entries := []trace.Event{
		{Duration: 1},
		{Duration: 1},
		{Duration: 1},
	}
	total := time.Duration(3)

	// exact shares are all 33.333...%, an exact three-way tie on the
	// remainder; the deterministic tiebreak is the original entry order.
	want := []int{34, 33, 33}
	got := distributePercentages(entries, total)
	if !slices.Equal(got, want) {
		t.Error(test.DiffMessage(got, want, "tied remainders must be broken deterministically by original order"))
	}
}

func TestDistributePercentages_Deterministic(t *testing.T) {
	entries := []trace.Event{
		{Duration: 7}, {Duration: 13}, {Duration: 5}, {Duration: 22}, {Duration: 9},
	}
	total := sumDurations(entries)
	first := distributePercentages(entries, total)
	second := distributePercentages(entries, total)
	if !slices.Equal(first, second) {
		t.Error(test.DiffMessage(second, first, "distributePercentages must return the same result for the same input"))
	}
}

func TestDistributePercentages_ZeroTotalReturnsZeros(t *testing.T) {
	entries := []trace.Event{{Duration: 0}, {Duration: 0}}
	got := distributePercentages(entries, 0)
	want := []int{0, 0}
	if !slices.Equal(got, want) {
		t.Error(test.DiffMessage(got, want, "zero total must not divide by zero and must return all-zero percentages"))
	}
}

func TestDistributePercentages_EmptyEntries(t *testing.T) {
	got := distributePercentages(nil, 100)
	if len(got) != 0 {
		t.Error(test.DiffMessage(got, "[]", "no entries must return an empty slice"))
	}
}

func TestDistributePercentages_OvershootSubtractsFromSmallestRemainderFirst(t *testing.T) {
	entries := []trace.Event{
		{Duration: 40},
		{Duration: 35},
		{Duration: 26},
	}
	// total is deliberately inconsistent with the entries' real sum (101),
	// forcing the exact shares to sum above 100 and exercising the
	// subtract-from-smallest-remainder branch.
	total := time.Duration(100)

	want := []int{39, 35, 26}
	got := distributePercentages(entries, total)
	if !slices.Equal(got, want) {
		t.Error(test.DiffMessage(got, want, "when floors overshoot 100, the entry with the smallest remainder must lose the point first"))
	}
	if sum := sumInts(got); sum != 100 {
		t.Error(test.DiffMessage(sum, 100, "percentages must still sum to exactly 100 after subtracting the overshoot"))
	}
}

func TestAccessLog_ConcurrentEmit_NoDataRace(t *testing.T) {
	logger := &fakeLogger{ch: make(chan recordedLog, 256)}
	ev := event.NewEvent()
	NewAccessLog(&AccessLogConfig{Event: ev, Logger: logger})

	const goroutines = 32
	const eventsPerID = 5

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := range goroutines {
		go func(id string) {
			defer wg.Done()
			for j := 0; j < eventsPerID; j++ {
				ev.Emit(trace.EventName, trace.Event{ID: id, Stage: trace.StageMiddleware, Name: fmt.Sprintf("M%d", j)})
			}
			ev.Emit(trace.EventName, trace.Event{ID: id, Stage: trace.StageComplete, Code: 200, Target: "Request"})
		}(fmt.Sprintf("req-%d", i))
	}
	wg.Wait()

	for i := 0; i < goroutines; i++ {
		recvLog(t, logger.ch)
	}
}
