package utils

import (
	"crypto/rand"
	"fmt"

	"github.com/dustin/randbo"
)

// RandomCreateBytes generate random []byte by specify chars.
//func RandomCreateBytes(n int, alphabets ...byte) []byte {
func RandomCreateBytes(n int, alphabets string) []byte {
	const alphanum = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	var bytes = make([]byte, n)
	rand.Read(bytes)
	for i, b := range bytes {
		if len(alphabets) == 0 {
			bytes[i] = alphanum[b%byte(len(alphanum))]
		} else {
			bytes[i] = alphabets[b%byte(len(alphabets))]
		}
	}
	return bytes
}

// fast random string
func FastRequestId(n int) string {
	buf := make([]byte, n)
	randbo.New().Read(buf) //号称最快的随机字符串
	return fmt.Sprintf("%x", buf)
}
