package main

import (
	"log"
	"net/http"
)

func healthzHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
	w.Write([]byte("OK"))
}

func main() {
	SM := http.NewServeMux()
	Server := &http.Server{Addr: ":8080", Handler: SM}
	SM.Handle("/app/", http.StripPrefix("/app", http.FileServer(http.Dir("."))))
	SM.HandleFunc("/healthz", healthzHandler)
	err := Server.ListenAndServe()
	if err != nil {
		log.Fatal("Error: unable to start server")
	}
}
