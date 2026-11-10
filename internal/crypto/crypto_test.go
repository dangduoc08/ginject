package crypto

import (
	"strings"
	"sync"
	"testing"

	"github.com/dangduoc08/ginject/internal/test"
)

func TestUUID_Format(t *testing.T) {
	id := UUID()
	if len(id) != 36 {
		t.Fatal(test.DiffMessage(len(id), 36, "UUID length"))
	}
	for _, i := range []int{8, 13, 18, 23} {
		if id[i] != '-' {
			t.Error(test.DiffMessage(string(id[i]), "-", "UUID "+id+" dash position"))
		}
	}
	for i, c := range id {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			continue
		}
		if !strings.ContainsRune("0123456789abcdef", c) {
			t.Error(test.DiffMessage(string(c), "lowercase hex", "UUID "+id+" character"))
		}
	}
}

func TestUUID_Version4(t *testing.T) {
	for range 1000 {
		id := UUID()
		if id[14] != '4' {
			t.Fatal(test.DiffMessage(string(id[14]), "4", "UUID "+id+" version nibble"))
		}
	}
}

func TestUUID_VariantRFC9562(t *testing.T) {
	for range 1000 {
		id := UUID()
		if !strings.ContainsRune("89ab", rune(id[19])) {
			t.Fatal(test.DiffMessage(string(id[19]), "one of 8 9 a b", "UUID "+id+" variant nibble"))
		}
	}
}

func TestUUID_ConcurrentUnique(t *testing.T) {
	const goroutines = 32
	const perGoroutine = 1000
	var mu sync.Mutex
	var wg sync.WaitGroup
	seen := make(map[string]struct{}, goroutines*perGoroutine)

	for range goroutines {
		wg.Go(func() {
			local := make([]string, 0, perGoroutine)
			for range perGoroutine {
				local = append(local, UUID())
			}
			mu.Lock()
			for _, id := range local {
				seen[id] = struct{}{}
			}
			mu.Unlock()
		})
	}

	wg.Wait()
	if len(seen) != goroutines*perGoroutine {
		t.Error(test.DiffMessage(len(seen), goroutines*perGoroutine, "distinct UUIDs across goroutines"))
	}
}
