package crypto

import (
	cryptoRand "crypto/rand"
	"encoding/hex"
)

func UUID() string {
	var uuid [16]byte
	_, _ = cryptoRand.Read(uuid[:])

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
	return string(buf[:])
}
