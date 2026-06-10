package main

import (
	"log"
	"mime"
	"net/http"
	"path/filepath"
	"runtime"
)

func main() {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		log.Fatal("could not resolve web directory")
	}
	webDir := filepath.Dir(file)

	_ = mime.AddExtensionType(".wasm", "application/wasm")

	fs := http.FileServer(http.Dir(webDir))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if filepath.Ext(r.URL.Path) == ".wasm" {
			w.Header().Set("Content-Type", "application/wasm")
		}
		fs.ServeHTTP(w, r)
	})

	addr := "localhost:8080"
	log.Printf("serving %s at http://%s", webDir, addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
