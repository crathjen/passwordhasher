package main

import (
	"context"
	"fmt"
	"net/http"
	"sync"
)

func main() {
	hashCount := &HashCount{}
	hashStats := &HashStats{}
	hashValues := NewHashValues()
	wg := &sync.WaitGroup{}
	shutdownCh := make(chan struct{})

	http.HandleFunc("/hash", statsCollector(hashStats, getHasherHandler(hashCount, hashValues, wg)))
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
