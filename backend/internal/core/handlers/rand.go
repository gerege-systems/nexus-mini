package handlers

import "crypto/rand"

func randByte() byte {
	var b [1]byte
	_, _ = rand.Read(b[:])
	return b[0]
}
