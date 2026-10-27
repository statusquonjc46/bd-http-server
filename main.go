package main

import (
	"fmt"
	"net/http"
)

func main() {
	mux := http.NewServeMux()
	httpServer := http.Server{
		Handler: mux,
		Addr:    ":8080",
	}
	rootDir := http.Dir(".")

	mux.Handle("/app/", http.StripPrefix("/app/", http.FileServer(rootDir)))
	mux.HandleFunc("/healthz", healthHandler)

	if err := httpServer.ListenAndServe(); err != nil {
		fmt.Println(fmt.Errorf("Error: failed to start server [%v]", err))
	}
}
