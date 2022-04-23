package main

import (
	"context"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

type HashCount struct {
	count int
	lock sync.Mutex
}

func (hc *HashCount) IncrementHashCount() int {
	hc.lock.Lock()
	defer hc.lock.Unlock()
	hc.count ++
	return hc.count
}

type Hash struct {
	ID int
	Value string
}

type HashStats struct {
	lock sync.RWMutex
	count int
	totalDurationInMicroseconds int64
}

func (hs *HashStats) AddRequestStats (duration int64) {
	hs.lock.Lock()
	defer hs.lock.Unlock()
	hs.count ++
	hs.totalDurationInMicroseconds += duration
}

func (hs *HashStats) GetStats() (int, int64) {
	hs.lock.RLock()
	defer hs.lock.RUnlock()
	if hs.count == 0 {
		return 0, 0
	}
	return hs.count, hs.totalDurationInMicroseconds/int64(hs.count)
}

type HashValues struct {
	values map[int]string
	lock sync.RWMutex
}

func NewHashValues() *HashValues {
	return &HashValues{values: make(map[int]string)}
}

func (hv *HashValues) AddHash(h Hash) {
	hv.lock.Lock()
	defer hv.lock.Unlock()
	hv.values[h.ID]=h.Value
}

func (hv *HashValues) GetHash(id int) (string, bool) {
	hv.lock.RLock()
	defer hv.lock.RUnlock()
	hash, found := hv.values[id]
	return hash, found
}


func getHasherHandler(hc *HashCount, hashCh chan<- Hash, wg *sync.WaitGroup) func (resp http.ResponseWriter, req *http.Request) {
	return func(resp http.ResponseWriter, req *http.Request) {
		req.ParseForm()

		pw := req.Form.Get("password")
		
		hashID := hc.IncrementHashCount()
		
		wg.Add(1)
		go func() {
			time.Sleep(5 * time.Second)
			h := sha512.New()
			h.Write([]byte(pw))
			hashCh <- Hash{Value: base64.StdEncoding.EncodeToString(h.Sum(nil)), ID: hashID}
		}()

		resp.Write([]byte(strconv.Itoa(hashID)))
	}
}

func statsCollector(hs *HashStats, wrappedHandlerFunc func (resp http.ResponseWriter, req *http.Request)) func (http.ResponseWriter, *http.Request) {
	return func (resp http.ResponseWriter, req *http.Request)  {
		start := time.Now()
		wrappedHandlerFunc(resp, req)
		hs.AddRequestStats(time.Since(start).Microseconds())
	}

	
}

type StatsResponseDTO struct {
	Total int `json:"total"`
	Average int64 `json:"average"`
}

func getStatsHandler(hs *HashStats) func (resp http.ResponseWriter, req *http.Request) {
	return func(resp http.ResponseWriter, req *http.Request) {
		total, avg := hs.GetStats()
		
		jsonResult, err := json.Marshal(StatsResponseDTO{Total: total, Average: avg})
		if err != nil {
			resp.WriteHeader(http.StatusInternalServerError)
			resp.Write([]byte("encountered an error"))
		}else {
			resp.Write(jsonResult)
		}

	}
}


func getShutdownHandler(shutdownCh chan<- struct{}) func (resp http.ResponseWriter, req *http.Request) {
	return func(resp http.ResponseWriter, req *http.Request) {
		resp.Write([]byte("shutting down"))
		shutdownCh <- struct{}{}
	}
}

func getHashHandler(hv *HashValues) func (resp http.ResponseWriter, req *http.Request) {
	return func(resp http.ResponseWriter, req *http.Request) {
		idString := strings.TrimPrefix(req.URL.Path, "/hash/")
		id, err := strconv.Atoi(idString)
		if err != nil {
			resp.WriteHeader(http.StatusBadRequest)
			resp.Write([]byte("bad request"))
		}else {
			hash, found := hv.GetHash(id)
			if !found {
				resp.WriteHeader(http.StatusNotFound)
				resp.Write([]byte("hash not found"))
			} else {
				resp.Write([]byte(hash))
			}
		}

	}
}


func main() {
	hashCount := &HashCount{}
	hashCh := make(chan Hash)
	hashStats := &HashStats{}
	hashValues := NewHashValues()
	wg := &sync.WaitGroup{}
	shutdownCh := make(chan struct{})

	go func() {
		for hash := range hashCh {
			hashValues.AddHash(hash)
			fmt.Println("hash added")
			wg.Done()
		}
	}()

	http.HandleFunc("/hash", statsCollector(hashStats, getHasherHandler(hashCount, hashCh, wg)))
	http.HandleFunc("/hash/", getHashHandler(hashValues))
	http.HandleFunc("/stats", getStatsHandler(hashStats))
	http.HandleFunc("/shutdown", getShutdownHandler(shutdownCh))

	server := &http.Server{Addr: ":8080"}
	go func() {
		<-shutdownCh
		err := server.Shutdown(context.Background())
		if err != nil {
			fmt.Printf("shutdown error: %v\n", err.Error())
		}
	}()

	if err := server.ListenAndServe(); err != nil {
		fmt.Printf("shutting down: %v\n", err.Error())
	}

	wg.Wait()
}