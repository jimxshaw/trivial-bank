package util

import (
	cryptoRand "crypto/rand"
	"encoding/hex"
	"math/rand"
	"strings"
	"time"
)

const alphabet = "abcdefghijklmnopqrstuvwxyz"

var r *rand.Rand

func init() {
	s := rand.NewSource(time.Now().UnixNano())
	r = rand.New(s)
}

// RandomInt generates a random int between min and max.
func RandomInt(min, max int64) int64 {
	return min + r.Int63n(max-min+1)
}

// RandomString generates a random string of n characters.
func RandomString(n int) string {
	var sb strings.Builder
	l := len(alphabet)

	for i := 0; i < n; i++ {
		char := alphabet[r.Intn(l)]
		sb.WriteByte(char)
	}

	return sb.String()
}

// RandomHex generates random hex values.
func RandomHex(n int) (string, error) {
	bytes := make([]byte, n)

	if _, err := cryptoRand.Read(bytes); err != nil {
		return "", err
	}

	return hex.EncodeToString(bytes), nil
}

// RandomOwner generates a random owner name.
func RandomOwner() string {
	return RandomString(8)
}

// RandomAmount generates a random amount of money.
func RandomAmount() int64 {
	return RandomInt(0, 1_000_000)
}

// RandomCurrency generates a random currency code.
func RandomCurrency() string {
	curr := []string{"USD", "EUR", "GBP", "CAD", "CNY", "AUD", "MXN"}
	l := len(curr)

	return curr[rand.Intn(l)]
}
