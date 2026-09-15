package utils

import (
	"crypto/rand"
	"math/big"
)

const randCharset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890"

// RandString 使用加密安全随机源生成指定长度的随机字符串。
func RandString(number int) string {
	if number <= 0 {
		number = 32
	}
	max := big.NewInt(int64(len(randCharset)))
	out := make([]byte, number)
	for i := range out {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			out[i] = randCharset[0]
			continue
		}
		out[i] = randCharset[n.Int64()]
	}
	return string(out)
}
