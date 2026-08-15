package templates

import (
	"crypto/rand"
	"fmt"
	"gorogs/logger"
	"os"
	"path/filepath"
	"strings"
)

var currentInstanceUUID string

func GenerateRandomUUIDv4() string {
	bytes := make([]byte, 16)
	_, err := rand.Read(bytes)
	if err != nil {
		logger.Error("wsdd", "Critical tracking error: system entropy pool is exhausted, falling back to dead static safety identifier", err)
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
	logger.DebugF("wsdd", "Cryptographically secure unique transaction identifier string generated successfully: %s", uuidStr)
	return uuidStr
}

func LoadOrCreatePersistentUUID(configDir string, serverName string) {
	cleanServerName := strings.ToLower(strings.TrimSpace(serverName))
	if cleanServerName == "" {
		logger.Error("wsdd", "Config parameter failure: empty ServerName supplied to identity loader, falling back to generic token mapping", nil)
		cleanServerName = "generic-node"
	}
	fileName := fmt.Sprintf("uuid-%s.txt", cleanServerName)
	filePath := filepath.Join(configDir, fileName)
	data, err := os.ReadFile(filePath)
	if err == nil {
		currentInstanceID := strings.TrimSpace(string(data))
		if len(currentInstanceID) == 36 {
			logger.DebugF("wsdd", "Successfully loaded persistent machine UUID from configuration storage for identity node '%s': %s", cleanServerName, currentInstanceID)
			return
		}
	}
	currentInstanceID := GenerateRandomUUIDv4()
	_ = os.MkdirAll(configDir, 0755)
	err = os.WriteFile(filePath, []byte(currentInstanceID), 0644)
	if err != nil {
		logger.ErrorF("wsdd", "Failed to write persistent machine UUID to shared volume configuration target path: %s", err, filePath)
	} else {
		logger.InfoF("wsdd", "Generated and securely saved fresh unique persistent machine identifier for identity node '%s': %s", cleanServerName, currentInstanceID)
	}
}
