package main

import (
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

func getHasherHandler(hc *HashCount, hv *HashValues, wg *sync.WaitGroup) http.HandlerFunc {
	return func(resp http.ResponseWriter, req *http.Request) {
		// assumes happy path due to time constraints - need to confirm http method, url path, content-type

		req.ParseForm()

		// this is optimistic and should be validated
		pw := req.Form.Get("password")

		hashID := hc.IncrementHashCount()

		wg.Add(1)
		go func() {
			defer wg.Done()
			time.Sleep(5 * time.Second)
			h := sha512.New()
			_, err := h.Write([]byte(pw))
			if err != nil {
				fmt.Printf("hashing error: %v", err.Error())
			} else {
				hv.AddHash(Hash{Value: base64.StdEncoding.EncodeToString(h.Sum(nil)), ID: hashID})
				fmt.Printf("password with id %d was hashed\n", hashID)
			}
		}()

		resp.Write([]byte(strconv.Itoa(hashID)))
	}
}

func statsCollector(hs *HashStats, wrappedHandlerFunc http.HandlerFunc) http.HandlerFunc {
	return func(resp http.ResponseWriter, req *http.Request) {
		start := time.Now()
		wrappedHandlerFunc(resp, req)
		hs.AddRequestStats(time.Since(start).Microseconds())
	}
}

func getStatsHandler(hs *HashStats) http.HandlerFunc {
	return func(resp http.ResponseWriter, req *http.Request) {
		// assumes happy path due to time constraints - need to confirm http method, url path

		total, avg := hs.GetStats()

		jsonResult, err := json.Marshal(StatsResponseDTO{Total: total, Average: avg})
		if err != nil {
			resp.WriteHeader(http.StatusInternalServerError)
			resp.Write([]byte("encountered an error"))
		} else {
			resp.Write(jsonResult)
		}

	}
}

func getShutdownHandler(shutdownCh chan<- struct{}) http.HandlerFunc {
	return func(resp http.ResponseWriter, req *http.Request) {
		resp.Write([]byte("shutting down"))
		shutdownCh <- struct{}{}
	}
}

func getHashHandler(hv *HashValues) http.HandlerFunc {
	return func(resp http.ResponseWriter, req *http.Request) {

		// assumes happy path due to time constraints - need to confirm http method, url path

		idString := strings.TrimPrefix(req.URL.Path, "/hash/")
		// this is brittle - using a muxing library to assist would be the best way to do this for production
		id, err := strconv.Atoi(idString)
		if err != nil {
			resp.WriteHeader(http.StatusBadRequest)
			resp.Write([]byte("bad request"))
		} else {
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
