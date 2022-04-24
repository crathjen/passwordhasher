package main

import "sync"

type HashCount struct {
	count int
	lock  sync.Mutex
}

func (hc *HashCount) IncrementHashCount() int {
	hc.lock.Lock()
	defer hc.lock.Unlock()
	hc.count++
	return hc.count
}

type Hash struct {
	ID    int
	Value string
}

type HashStats struct {
	lock                        sync.RWMutex
	count                       int
	totalDurationInMicroseconds int64
}

func (hs *HashStats) AddRequestStats(duration int64) {
	hs.lock.Lock()
	defer hs.lock.Unlock()
	hs.count++
	hs.totalDurationInMicroseconds += duration
}

func (hs *HashStats) GetStats() (int, int64) {
	hs.lock.RLock()
	defer hs.lock.RUnlock()
	if hs.count == 0 {
		return 0, 0
	}
	return hs.count, hs.totalDurationInMicroseconds / int64(hs.count)
}

type HashValues struct {
	values map[int]string
	lock   sync.RWMutex
}

func NewHashValues() *HashValues {
	return &HashValues{values: make(map[int]string)}
}

func (hv *HashValues) AddHash(h Hash) {
	hv.lock.Lock()
	defer hv.lock.Unlock()
	hv.values[h.ID] = h.Value
}

func (hv *HashValues) GetHash(id int) (string, bool) {
	hv.lock.RLock()
	defer hv.lock.RUnlock()
	hash, found := hv.values[id]
	return hash, found
}

type StatsResponseDTO struct {
	Total   int   `json:"total"`
	Average int64 `json:"average"`
}
