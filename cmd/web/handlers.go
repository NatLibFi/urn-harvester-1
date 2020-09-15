package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/wvh/urn/internal/version"
)

func handleVersion() http.HandlerFunc {
	response, err := json.Marshal(
		struct {
			Id      string `json:"id"`
			Name    string `json:"name,omitempty"`
			Tag     string `json:"tag,omitempty"`
			Hash    string `json:"hash,omitempty"`
			Branch  string `json:"branch,omitempty"`
			Repo    string `json:"repo,omitempty"`
			Version string `json:"version"`
		}{
			version.Id,
			version.Name,
			version.Tag,
			version.Hash,
			version.Branch,
			version.Repo,
			version.Version,
		},
	)
	if err != nil {
		// if we can't even encode static strings, choose to panic
		panic(err)
	}

	plain := []byte(version.Id + " " + version.Version + "\n")

	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Vary", "Accept")

		if strings.HasPrefix(r.Header.Get("Accept"), "text/plain") {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Write(plain)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(response)
	}
}

func helloHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("hello"))
}

// var healthy int64

/*
func (c *controller) healthz(w http.ResponseWriter, req *http.Request) {
	if h := atomic.LoadInt64(&c.healthy); h == 0 {
		w.WriteHeader(http.StatusServiceUnavailable)
	} else {
		fmt.Fprintf(w, "uptime: %s\n", time.Since(time.Unix(0, h)))
	}
}
*/

func logMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("request: %s %s\n", r.Method, r.URL.Path)
		log.Printf("  %+v\n", r)
		log.Printf("  %#v\n", r.URL)
		next.ServeHTTP(w, r)
	})
}

var (
	errNoHostConfigured = errors.New("I don't serve anything on that hostname")
)

func authoritiveHostOnly(hn string, next http.Handler) http.Handler {
	if hn == "*" {
		return next
	}

	if hn == "" {
		sys, err := os.Hostname()
		if err != nil {
			panic(err)
		}
		hn = sys
	}

	log.Println("only listening for requests on", hn)

	localIP4 := "127.0.0.1" + ":" + httpPort
	localIP6 := "[::1]" + ":" + httpPort

	// add port to hostname for non-standard ports
	if httpPort != "80" && httpPort != "443" {
		hn = hn + ":" + httpPort
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.EqualFold(hn, r.Host) && r.Host != localIP4 && r.Host != localIP6 {
			log.Printf("bad host: %s\n", r.Host)
			http.Error(w, errNoHostConfigured.Error(), http.StatusBadRequest)
			return
		}
		next.ServeHTTP(w, r)
	})
}
