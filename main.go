package main

import (
	"fmt"
	"net/http"
)

// create rootdir constant so it cannot be changed
const rootDir = http.Dir(".")

func main() {
	//instantiate apiConfig struct to track metrics
	apiCfg := apiConfig{}

	//instantiate a new server mux to handle all http requests
	mux := http.NewServeMux()
	//instantiate a new http server
	httpServer := http.Server{
		Handler: mux,
		Addr:    ":8080",
	}

	//handle /app/ path 127.0.0.1:8080/app/, string /app/ so that the file path is used without /app/ to serve from the file path
	// apiCfg.middlewareMetricsIncrement wraps the http.FileServer to count the hits to /app/
	mux.Handle("/app/", http.StripPrefix("/app/", apiCfg.middlewareMetricsIncrement(http.FileServer(rootDir))))

	//handles requests to check site up/down, returns page with "OK"
	mux.HandleFunc("GET /healthz", healthHandler)

	//handles requests to see the hits to the app, returns "Hits: x"
	mux.HandleFunc("GET /metrics", apiCfg.metricHandler)

	//handles requests to reset the site metrics
	mux.HandleFunc("POST /reset", apiCfg.resetHandler)

	//listen and serve sets up the interface to take requests and return responses based on the handlers above
	if err := httpServer.ListenAndServe(); err != nil {
		fmt.Println(fmt.Errorf("Error: failed to start server [%v]", err))
	}
}
