package netgear

import (
	"crypto/md5"
	"encoding/hex"
)

// mergeHash matches py-netgear-plus: MD5 of interleaved password+rand.
func mergeHash(password, rand string) string {
	a, b := []rune(password), []rune(rand)
	out := make([]rune, 0, len(a)+len(b))
	for i, j := 0, 0; i < len(a) || j < len(b); i, j = i+1, j+1 {
		if i < len(a) {
			out = append(out, a[i])
		}
		if j < len(b) {
			out = append(out, b[j])
		}
	}
	sum := md5.Sum([]byte(string(out)))
	return hex.EncodeToString(sum[:])
}
