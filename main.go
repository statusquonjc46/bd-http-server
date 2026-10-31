package main

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/statusquonjc46/bd-http-server/internal/database"
)

// create rootdir constant so it cannot be changed
const rootDir = http.Dir(".")

func main() {
	//env variable instantiation
	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	platform := os.Getenv("PLATFORM")
	//Open connection to psql db
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		fmt.Println(errors.New("Failed to open connection to database"))
		os.Exit(1)
	}
	//instantiate apiConfig struct to track metrics
	apiCfg := apiConfig{}
	apiCfg.database = database.New(db)
	apiCfg.platform = platform
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
	mux.HandleFunc("GET /api/healthz", healthHandler)

	//handles requests to see the hits to the app, returns "Hits: x"
	mux.HandleFunc("GET /admin/metrics", apiCfg.metricHandler)

	//handles requests to reset the site metrics
	mux.HandleFunc("POST /admin/reset", apiCfg.resetHandler)

	//handles requests to validate a chirp
	mux.HandleFunc("POST /api/validate_chirp", validateChirpHandler)

	//handles adding users to the users table in the chirpy db
	mux.HandleFunc("POST /api/users", apiCfg.addUserHandler)

	//listen and serve sets up the interface to take requests and return responses based on the handlers above
	if err := httpServer.ListenAndServe(); err != nil {
		fmt.Println(fmt.Errorf("Error: failed to start server [%v]", err))
	}
}
