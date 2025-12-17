package random

import (
	"crypto/rand"
	"encoding/hex"
)

// generate random url using rand package
func GenerateRandomUrl() string {
	randomUrl := make([]byte, 10) // for now buffer size is fixed size
	rand.Read(randomUrl)          // Read function returns an error but it's always nil

	encodedUrl := hex.EncodeToString(randomUrl)
	return encodedUrl
}
