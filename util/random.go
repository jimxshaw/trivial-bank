package util

import (
	cryptoRand "crypto/rand"
	"encoding/hex"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/jimxshaw/trivial-bank/util/currency"
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

// RandomAmount generates a random amount of money.
func RandomAmount() int64 {
	return RandomInt(0, 1_000_000)
}

// RandomCurrency generates a random currency code.
func RandomCurrency() string {
	curr := []string{
		currency.USD,
		currency.EUR,
		currency.GBP,
		currency.CAD,
		currency.CNY,
		currency.AUD,
		currency.MXN,
	}
	l := len(curr)

	return curr[rand.Intn(l)]
}

// RandomPassword generates a random hashed password.
func RandomPassword() (string, error) {
	password := RandomString(15)
	hash, err := HashPassword(password)
	if err != nil {
		return "", err
	}

	return hash, nil
}

// RandomEmail generates a random email address.
func RandomEmail() string {
	username := RandomString(10)
	domain := RandomString(10)
	extension := RandomString(3)

	return fmt.Sprintf("%s@%s.%s", username, domain, extension)
}

// RandomUsernam generates a random username.
func RandomUsername() string {
	prefix := RandomString(10)
	suffix := RandomInt(0, 99)

	return fmt.Sprintf("%s%d", prefix, suffix)
}
