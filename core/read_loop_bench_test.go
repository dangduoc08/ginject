package core

import "testing"

func BenchmarkHandlePublish_DeliversAfterSubscribe(b *testing.B) {
	ws := newTestWSBare(b)
	ws.eventMatcher.AddInjectableHandler("chat.to.*", func() {})

	serverConn, clientConn, cleanup := newTestWSConnPair(b)
	defer cleanup()

	conn := ws.connmgr.Register("conn-1", serverConn)
	defer ws.connmgr.Unregister("conn-1")

	handleSubscribe(conn, ws, WSPayload{ID: "req-1", Type: TypeSubscribe, Topic: []string{"chat.to.user2"}})
	recvWSPayload(b, clientConn)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handlePublish(conn, ws, WSPayload{ID: "req-2", Type: TypePublish, Topic: []string{"chat.to.user2"}, Message: "hi"})

		recvWSPayload(b, clientConn)
		recvWSPayload(b, clientConn)
	}
}
