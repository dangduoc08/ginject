package exception

import "testing"

func BenchmarkTopicNotFoundException(b *testing.B) {
	for range b.N {
		_ = TopicNotFoundException("no handler registered for topic: chat.to.user1")
	}
}

func BenchmarkPolicyViolationException(b *testing.B) {
	for range b.N {
		_ = PolicyViolationException("policy violation")
	}
}

func BenchmarkWSInternalErrorException(b *testing.B) {
	for range b.N {
		_ = WSInternalErrorException("internal error")
	}
}
