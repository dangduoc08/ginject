package httpclient

import (
	"crypto/tls"
	"net/http/httptrace"
	"sync/atomic"
	"time"
)

// TimingInfo records per-phase durations for a single HTTP request.
type TimingInfo struct {
	DNS   time.Duration
	TCP   time.Duration
	TLS   time.Duration
	TTFB  time.Duration
	Total time.Duration
}

// timingCollector uses atomic int64 (UnixNano) so that Happy-Eyeballs parallel
// dial goroutines can write fields concurrently without a race.
type timingCollector struct {
	startNs     int64 // set once at creation, never changes
	dnsStartNs  int64
	dnsDoneNs   int64
	tcpStartNs  int64
	tcpDoneNs   int64
	tlsStartNs  int64
	tlsDoneNs   int64
	firstByteNs int64
}

func newTimingCollector() *timingCollector {
	return &timingCollector{startNs: time.Now().UnixNano()}
}

func (tc *timingCollector) clientTrace() *httptrace.ClientTrace {
	now := func() int64 { return time.Now().UnixNano() }
	return &httptrace.ClientTrace{
		DNSStart:             func(_ httptrace.DNSStartInfo) { atomic.StoreInt64(&tc.dnsStartNs, now()) },
		DNSDone:              func(_ httptrace.DNSDoneInfo) { atomic.StoreInt64(&tc.dnsDoneNs, now()) },
		ConnectStart:         func(_, _ string) { atomic.StoreInt64(&tc.tcpStartNs, now()) },
		ConnectDone:          func(_, _ string, _ error) { atomic.StoreInt64(&tc.tcpDoneNs, now()) },
		TLSHandshakeStart:    func() { atomic.StoreInt64(&tc.tlsStartNs, now()) },
		TLSHandshakeDone:     func(_ tls.ConnectionState, _ error) { atomic.StoreInt64(&tc.tlsDoneNs, now()) },
		GotFirstResponseByte: func() { atomic.StoreInt64(&tc.firstByteNs, now()) },
	}
}

func (tc *timingCollector) build() *TimingInfo {
	endNs := time.Now().UnixNano()
	info := &TimingInfo{Total: time.Duration(endNs - tc.startNs)}

	if d := atomic.LoadInt64(&tc.dnsDoneNs); d != 0 {
		info.DNS = time.Duration(d - atomic.LoadInt64(&tc.dnsStartNs))
	}
	if d := atomic.LoadInt64(&tc.tcpDoneNs); d != 0 {
		info.TCP = time.Duration(d - atomic.LoadInt64(&tc.tcpStartNs))
	}
	if d := atomic.LoadInt64(&tc.tlsDoneNs); d != 0 {
		info.TLS = time.Duration(d - atomic.LoadInt64(&tc.tlsStartNs))
	}
	if fb := atomic.LoadInt64(&tc.firstByteNs); fb != 0 {
		info.TTFB = time.Duration(fb - tc.startNs)
	}
	return info
}
