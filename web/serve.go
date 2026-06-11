// Serve intertui web UI with a Socket.IO reverse proxy to the game server.
//
// Usage:
//
//	./web/build.sh
//	go run ./web/serve.go
//	open 'http://localhost:8080/?server=GAME_HOST&user=YOU&pass=SECRET'
//
// Set INTERTUI_PROXY=http://game:13370 to pin the upstream without ?server= in the URL.
package main

import (
	"flag"
	"log"
	"mime"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

const defaultSIOPort = 13370

func main() {
	addr := flag.String("addr", "localhost:8080", "listen address")
	flag.Parse()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		log.Fatal("could not resolve web directory")
	}
	webDir := filepath.Dir(file)

	_ = mime.AddExtensionType(".wasm", "application/wasm")

	fs := http.FileServer(http.Dir(webDir))
	mux := http.NewServeMux()
	mux.HandleFunc("/socket.io/", proxySocketIO)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if filepath.Ext(r.URL.Path) == ".wasm" {
			w.Header().Set("Content-Type", "application/wasm")
		}
		fs.ServeHTTP(w, r)
	})

	log.Printf("serving %s at http://%s", webDir, *addr)
	log.Printf("Socket.IO proxied at /socket.io/ (use ?server=HOST in page URL, or INTERTUI_PROXY)")
	log.Fatal(http.ListenAndServe(*addr, mux))
}

func proxySocketIO(w http.ResponseWriter, r *http.Request) {
	target, err := upstreamTarget(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	if target == "" {
		http.Error(w, "no upstream: set INTERTUI_PROXY or open the app with ?server=HOST", http.StatusBadGateway)
		return
	}

	u, err := url.Parse(target)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	proxy := httputil.NewSingleHostReverseProxy(u)
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("proxy %s %s: %v", target, r.URL.Path, err)
		http.Error(w, err.Error(), http.StatusBadGateway)
	}
	proxy.ServeHTTP(w, r)
}

func upstreamTarget(r *http.Request) (string, error) {
	if v := strings.TrimSpace(os.Getenv("INTERTUI_PROXY")); v != "" {
		return v, nil
	}

	ref := r.Referer()
	if ref == "" {
		return "", nil
	}
	page, err := url.Parse(ref)
	if err != nil {
		return "", err
	}
	q := page.Query()
	if q.Get("direct") == "1" || q.Get("direct") == "true" {
		return "", nil
	}

	server := q.Get("server")
	if server == "" {
		return "", nil
	}

	port := defaultSIOPort
	if p := q.Get("port"); p != "" {
		n, err := strconv.Atoi(p)
		if err != nil {
			return "", err
		}
		port = n
	}

	scheme := "http"
	if q.Get("tls") == "1" || q.Get("tls") == "true" {
		scheme = "https"
	}
	return scheme + "://" + net.JoinHostPort(server, strconv.Itoa(port)), nil
}
