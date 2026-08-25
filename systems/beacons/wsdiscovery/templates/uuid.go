package templates

import (
	"crypto/rand"
	"fmt"
)

var currentInstanceUUID string

func GenerateRandomUUIDv4() string {
	bytes := make([]byte, 16)
	_, err := rand.Read(bytes)
	if err != nil {
		return "00000000-0000-4000-8000-000000000000"
	}
	bytes[6] = (bytes[6] & 0x0f) | 0x40
	bytes[8] = (bytes[8] & 0x3f) | 0x80
	uuidStr := fmt.Sprintf("%x-%x-%x-%x-%x",
		bytes[0:4],
		bytes[4:6],
		bytes[6:8],
		bytes[8:10],
		bytes[10:16],
	)
	return uuidStr
}

func LoadOrCreatePersistentUUID() {
	if currentInstanceUUID != "" {
		return
	}
	currentInstanceUUID = GenerateRandomUUIDv4()
}
