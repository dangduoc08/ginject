package accesslog

import (
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dangduoc08/ginject/common"
	"github.com/dangduoc08/ginject/event"
	"github.com/dangduoc08/ginject/log"
	"github.com/dangduoc08/ginject/trace"
)

type AccessLogConfig struct {
	Event  *event.Event
	Logger common.Logger
}

type AccessLog struct {
	event  *event.Event
	logger common.Logger

	mu  sync.Mutex
	buf map[string][]trace.Event

	flush chan flushJob
}

type flushJob struct {
	done    trace.Event
	entries []trace.Event
}

func NewAccessLog(cfg *AccessLogConfig) *AccessLog {
	al := &AccessLog{
		event:  cfg.Event,
		logger: cfg.Logger,
		buf:    make(map[string][]trace.Event),
		flush:  make(chan flushJob, 1024),
	}

	go al.worker()

	al.event.On(trace.EventName, func(args ...any) {
		t := args[0].(trace.Event)

		al.mu.Lock()
		if t.Stage == trace.StageComplete {
			entries := al.buf[t.ID]
			delete(al.buf, t.ID)
			al.mu.Unlock()

			select {
			case al.flush <- flushJob{done: t, entries: entries}:
			default:
			}
			return
		}
		al.buf[t.ID] = append(al.buf[t.ID], t)
		al.mu.Unlock()
	})

	return al
}

func (al *AccessLog) worker() {
	for job := range al.flush {
		total := sumDurations(job.entries)
		stages := stageArgs(job.entries, total)
		args := make([]any, 0, 10+len(stages))
		args = append(args,
			"id", job.done.ID,
			"transport", job.done.Transport,
			"code", job.done.Code,
			"operation", job.done.Operation,
		)
		args = append(args, stages...)
		args = append(args, "total", formatDuration(total))
		al.logger.Info(job.done.Target, args...)
	}
}

func sumDurations(entries []trace.Event) time.Duration {
	var total time.Duration
	for _, e := range entries {
		total += e.Duration
	}
	return total
}

func stageArgs(entries []trace.Event, total time.Duration) []any {
	percents := distributePercentages(entries, total)

	byStage := make(map[string][]map[string]log.Colored, len(entries))
	order := make([]string, 0, len(entries))

	for i, e := range entries {
		if _, ok := byStage[e.Stage]; !ok {
			order = append(order, e.Stage)
		}
		byStage[e.Stage] = append(byStage[e.Stage], map[string]log.Colored{
			shortName(e.Name): {
				Text:  entryText(e.Duration, percents[i]),
				Color: colorForDuration(e.Duration),
			},
		})
	}

	args := make([]any, 0, len(order)*2)
	for _, stage := range order {
		args = append(args, stage, byStage[stage])
	}
	return args
}

// distributePercentages applies the Largest Remainder (Hamilton) method so
// the returned integer percentages always sum to exactly 100, instead of
// rounding each entry's share independently and letting the sum drift to
// 99 or 101.
func distributePercentages(entries []trace.Event, total time.Duration) []int {
	n := len(entries)
	percents := make([]int, n)
	if total <= 0 || n == 0 {
		return percents
	}

	remainders := make([]float64, n)
	sumFloor := 0
	for i, e := range entries {
		exact := float64(e.Duration) / float64(total) * 100
		floor := int(exact)
		percents[i] = floor
		remainders[i] = exact - float64(floor)
		sumFloor += floor
	}

	diff := 100 - sumFloor
	if diff == 0 {
		return percents
	}

	order := make([]int, n)
	for i := range order {
		order[i] = i
	}

	if diff > 0 {
		sort.SliceStable(order, func(a, b int) bool {
			return remainders[order[a]] > remainders[order[b]]
		})
		for i := 0; i < diff && i < n; i++ {
			percents[order[i]]++
		}
		return percents
	}

	sort.SliceStable(order, func(a, b int) bool {
		return remainders[order[a]] < remainders[order[b]]
	})
	for i := 0; i < -diff && i < n; i++ {
		percents[order[i]]--
	}
	return percents
}

func entryText(d time.Duration, percent int) string {
	buf := make([]byte, 0, 24)
	buf = append(buf, formatDuration(d)...)
	buf = append(buf, " ("...)
	buf = strconv.AppendInt(buf, int64(percent), 10)
	buf = append(buf, "%)"...)
	return string(buf)
}

func formatDuration(d time.Duration) string {
	switch {
	case d < time.Microsecond:
		return d.String()
	case d < time.Millisecond:
		return d.Round(time.Microsecond / 100).String()
	case d < time.Second:
		return d.Round(time.Millisecond / 100).String()
	default:
		return d.Round(time.Second / 100).String()
	}
}

func colorForDuration(d time.Duration) log.Color {
	switch {
	case d < 10*time.Millisecond:
		return log.ColorGray
	case d < 50*time.Millisecond:
		return log.ColorGreen
	case d < 100*time.Millisecond:
		return log.ColorBlue
	case d < 300*time.Millisecond:
		return log.ColorYellow
	case d < 1000*time.Millisecond:
		return log.ColorOrange
	default:
		return log.ColorRed
	}
}

func shortName(name string) string {
	if i := strings.LastIndex(name, "."); i >= 0 {
		return name[i+1:]
	}
	return name
}
