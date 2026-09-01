package connection

import (
	"hash/crc32"
	"sync"
	"time"
)

type cacheEntry struct {
	expiresAt time.Time
}

type MsgDeduplicator struct {
	mu            sync.Mutex
	store         map[uint32]cacheEntry
	lastCleaned   time.Time
	cleanInterval time.Duration
}

var GlobalPacketCache = &MsgDeduplicator{
	store:         make(map[uint32]cacheEntry, 256),
	lastCleaned:   time.Now(),
	cleanInterval: 30 * time.Second,
}

func (d *MsgDeduplicator) IsDuplicate(rawData []byte, retentionWindow time.Duration) bool {
	if len(rawData) == 0 {
		return false
	}

	checksum := crc32.ChecksumIEEE(rawData)
	now := time.Now()

	d.mu.Lock()
	defer d.mu.Unlock()

	if now.Sub(d.lastCleaned) > d.cleanInterval {
		for hashKey, entry := range d.store {
			if now.After(entry.expiresAt) {
				delete(d.store, hashKey)
			}
		}
		d.lastCleaned = now
	}

	if entry, found := d.store[checksum]; found {
		if now.Before(entry.expiresAt) {
			return true
		}
	}

	d.store[checksum] = cacheEntry{
		expiresAt: now.Add(retentionWindow),
	}

	return false
}
