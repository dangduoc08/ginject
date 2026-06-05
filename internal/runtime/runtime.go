package runtime

import (
	"os"
	"strconv"
	"time"
)

var nodeID = func() string {
	host, _ := os.Hostname()
	buf := make([]byte, 0, len(host)+1+10+1+20)
	buf = append(buf, host...)
	buf = append(buf, ':')
	buf = strconv.AppendInt(buf, int64(os.Getpid()), 10)
	buf = append(buf, ':')
	buf = strconv.AppendInt(buf, time.Now().UnixNano(), 10)
	return string(buf)
}()

func NodeID() string { return nodeID }
