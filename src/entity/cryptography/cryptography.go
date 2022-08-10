package cryptography

import (
	"crypto/md5"
	"crypto/sha1"
	"encoding/hex"
)

func MakeMD5(input string) string {
	hash := md5.Sum([]byte(input))
	return hex.EncodeToString(hash[:])
}

func MakeSHA1(input string) string {
	hash := sha1.Sum([]byte(input))
	return hex.EncodeToString(hash[:])
}
