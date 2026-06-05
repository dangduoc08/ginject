package crypto

import (
	cryptoRand "crypto/rand"
	"encoding/hex"
)

func withCharset(length int, charset string) string {
	b := make([]byte, length)
	if _, err := cryptoRand.Read(b); err != nil {
		panic(err)
	}
	n := len(charset)
	for i, rb := range b {
		b[i] = charset[int(rb)%n]
	}
	return string(b)
}

func Random(length int) string {
	charset := "`~1!2@3#4$5%6^7&8*9(0)-_=+qQwWeErRtTyYuUiIoOpP[{]}\\|aAsSdDfFgGhHjJkKlL;:'zZxXcCvVbBnNmM,<.>/?"
	return withCharset(length, charset)
}

func UUID() (string, error) {
	var uuid [16]byte
	if _, err := cryptoRand.Read(uuid[:]); err != nil {
		return "", err
	}

	uuid[6] = (uuid[6] & 0x0f) | 0x40
	uuid[8] = (uuid[8] & 0x3f) | 0x80

	var buf [36]byte
	hex.Encode(buf[0:8], uuid[0:4])
	buf[8] = '-'
	hex.Encode(buf[9:13], uuid[4:6])
	buf[13] = '-'
	hex.Encode(buf[14:18], uuid[6:8])
	buf[18] = '-'
	hex.Encode(buf[19:23], uuid[8:10])
	buf[23] = '-'
	hex.Encode(buf[24:36], uuid[10:16])
	return string(buf[:]), nil
}
