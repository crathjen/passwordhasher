package main

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"
)

// need a lot more test coverage for production code
func TestHasherHandler(t *testing.T) {
	hashCount := &HashCount{count: 2}
	hashValues := NewHashValues()
	wg := &sync.WaitGroup{}
	h := getHasherHandler(hashCount, hashValues, wg)

	data := url.Values{}
	data.Set("password", "angryMonkey")
	req, err := http.NewRequest(http.MethodPost, "/foo", strings.NewReader(data.Encode()))
	if err != nil {
		t.Fatalf("failed to create request: %v", err.Error())
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	recorder := httptest.NewRecorder()
	h.ServeHTTP(recorder, req)

	body, err := ioutil.ReadAll(recorder.Body)
	// definitely not fluent in testing with stdlib - usually use https://github.com/stretchr/testify
	if err != nil {
		t.Fatalf("failed to read response body: %v", err.Error())
	}
	if string(body) != "3" {
		t.Error("unexpected body")
	}
	if _, found := hashValues.GetHash(3); found {
		t.Error("hash should not be immediately available")
	}
	//hate having sleeps like this in tests - probably should make the sleep configurable
	time.Sleep(6 * time.Second)
	if hash, found := hashValues.GetHash(3); !found {
		t.Error("should have a hash after 6 seconds")
	} else if hash != "ZEHhWB65gUlzdVwtDQArEyx+KVLzp/aTaRaPlBzYRIFj6vjFdqEb0Q5B8zVKCZ0vKbZPZklJz0Fd7su2A+gf7Q==" {
		t.Errorf("hash value is not correct: %v", hash)
	}
}
