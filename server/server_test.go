// Copyright 2014 The Cockroach Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied.  See the License for the specific language governing
// permissions and limitations under the License. See the AUTHORS file
// for names of contributors.

package server

import (
	"compress/gzip"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

var (
	serverAddr string
	once       sync.Once
)

func startServer() {
	server := httptest.NewServer(New())
	serverAddr = server.Listener.Addr().String()
	log.Println("Test server listening on", serverAddr)
}

// TestHealthz verifies that /healthz does, in fact, return "ok"
// as expected.
func TestHealthz(t *testing.T) {
	once.Do(startServer)
	addr := "http://" + serverAddr + "/healthz"
	resp, err := http.Get(addr)
	if err != nil {
		t.Fatalf("error requesting healthz at %s: %s", addr, err)
	}
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("could not read response body: %s", err)
	}
	expected := "ok"
	if !strings.Contains(string(b), expected) {
		t.Errorf("expected body to contain %q, got %q", expected, string(b))
	}
}

// TestGzip hits the /healthz endpoint while explicitly disabling
// decompression on a custom client’s Transport and setting it
// conditionally via the request’s Accept-Encoding headers.
func TestGzip(t *testing.T) {
	once.Do(startServer)
	client := http.Client{
		Transport: &http.Transport{
			Proxy:              http.ProxyFromEnvironment,
			DisableCompression: true,
		},
	}
	req, err := http.NewRequest("GET", "http://"+serverAddr+"/healthz", nil)
	if err != nil {
		t.Fatalf("could not create request: %s", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("could not make request to %s: %s", req.URL, err)
	}
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("could not read response body: %s", err)
	}
	expected := "ok"
	if !strings.Contains(string(b), expected) {
		t.Errorf("expected body to contain %q, got %q", expected, string(b))
	}
	// Test for gzip explicitly.
	req.Header.Set("Accept-Encoding", "gzip")
	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("could not make request to %s: %s", req.URL, err)
	}
	defer resp.Body.Close()
	gz, err := gzip.NewReader(resp.Body)
	if err != nil {
		t.Fatalf("could not create new gzip reader: %s", err)
	}
	b, err = ioutil.ReadAll(gz)
	if err != nil {
		t.Fatalf("could not read gzipped response body: %s", err)
	}
	if !strings.Contains(string(b), expected) {
		t.Errorf("expected body to contain %q, got %q", expected, string(b))
	}
}
