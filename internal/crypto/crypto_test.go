package crypto

import (
	"strings"
	"testing"
)

func TestRandom_Length(t *testing.T) {
	for _, length := range []int{0, 1, 8, 32, 129} {
		s := Random(length)
		if len(s) != length {
			t.Errorf("Random(%d) length = %d, want %d", length, len(s), length)
		}
	}
}

func TestRandom_OnlyCharsetBytes(t *testing.T) {
	const charset = "`~1!2@3#4$5%6^7&8*9(0)-_=+qQwWeErRtTyYuUiIoOpP[{]}\\|aAsSdDfFgGhHjJkKlL;:'zZxXcCvVbBnNmM,<.>/?"
	s := Random(4096)
	for i, r := range s {
		if !strings.ContainsRune(charset, r) {
			t.Fatalf("Random output byte %d = %q, not in charset", i, r)
		}
	}
}

func TestRandom_Distinct(t *testing.T) {
	a := Random(32)
	b := Random(32)
	if a == b {
		t.Error("two Random(32) calls produced identical output; randomness looks broken")
	}
}

func TestWithCharset_PowerOfTwoCharsetLength(t *testing.T) {
	s := withCharset(64, "0123456789abcdef")
	if len(s) != 64 {
		t.Errorf("withCharset with an evenly-dividing charset length = %d, want 64", len(s))
	}
}

func TestUUID_Format(t *testing.T) {
	id, err := UUID()
	if err != nil {
		t.Fatal(err)
	}
	if len(id) != 36 {
		t.Fatalf("UUID length = %d, want 36", len(id))
	}
	if id[8] != '-' || id[13] != '-' || id[18] != '-' || id[23] != '-' {
		t.Errorf("UUID %q does not have dashes at the expected positions", id)
	}
	if id[14] != '4' {
		t.Errorf("UUID %q is not version 4 (expected '4' at position 14)", id)
	}
}
