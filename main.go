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
	bearer := os.Getenv("BEAR_SECRET")
	polka_key := os.Getenv("POLKA_KEY")
	//Open connection to psql db
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		fmt.Println(errors.New("Failed to open connection to database"))
		os.Exit(1)
	}
	//instantiate apiConfig struct to track metrics
	apiCfg := apiConfig{
		database: database.New(db),
		platform: platform,
		bearer:   bearer,
		polka:    polka_key,
	}

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

	//handles requests to get all chirps sorted by created-time
	mux.HandleFunc("GET /api/chirps", apiCfg.getAllChirpsHandler)

	//handles getting a chirp by chirp id
	mux.HandleFunc("GET /api/chirps/{chirpId}", apiCfg.getAChirp)

	//handles deleting a chirp if authenticated user
	mux.HandleFunc("DELETE /api/chirps/{chirpId}", apiCfg.deleteChirp)

	//handles requests to reset the site metrics
	mux.HandleFunc("POST /admin/reset", apiCfg.resetHandler)

	//handles requests to validate a chirp
	mux.HandleFunc("POST /api/chirps", apiCfg.chirpHandler)

	//handles adding users to the users table in the chirpy db
	mux.HandleFunc("POST /api/users", apiCfg.addUserHandler)

	//handles login requests
	mux.HandleFunc("POST /api/login", apiCfg.loginHandler)

	//handles refresh tokens
	mux.HandleFunc("POST /api/refresh", apiCfg.refreshHandler)

	//handles revoking the refresh token
	mux.HandleFunc("POST /api/revoke", apiCfg.revokeRefreshHandler)

	//handles updating authenticated users email and password
	mux.HandleFunc("PUT /api/users", apiCfg.updateProfileHandler)

	//handles upgrading a user to chirpy red
	mux.HandleFunc("POST /api/polka/webhooks", apiCfg.updateChirpyRed)

	//listen and serve sets up the interface to take requests and return responses based on the handlers above
	if err := httpServer.ListenAndServe(); err != nil {
		fmt.Println(fmt.Errorf("Error: failed to start server [%v]", err))
	}
}
