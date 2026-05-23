package httpclient

import "time"

type retryConfig struct {
	count       int
	initialWait time.Duration
	maxWait     time.Duration
}

func defaultRetryCondition(resp *Response, err error) bool {
	if err != nil {
		return true
	}
	return resp != nil && resp.StatusCode >= 500
}

func (rc retryConfig) execute(fn func() (*Response, error)) (*Response, error) {
	var (
		resp *Response
		err  error
		wait = rc.initialWait
	)
	for attempt := 0; attempt <= rc.count; attempt++ {
		resp, err = fn()
		if !defaultRetryCondition(resp, err) || attempt == rc.count {
			break
		}
		if wait > 0 {
			time.Sleep(wait)
			wait *= 2
			if rc.maxWait > 0 && wait > rc.maxWait {
				wait = rc.maxWait
			}
		}
	}
	return resp, err
}
