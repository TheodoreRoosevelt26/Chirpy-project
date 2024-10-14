package main

import (
	"log"
	"net/http"
)

func main() {
	SM := http.NewServeMux()
	Server := &http.Server{Addr: ":8080", Handler: SM}
	err := Server.ListenAndServe()
	if err != nil {
		log.Fatal("Error: unable to start server")
	}
}
