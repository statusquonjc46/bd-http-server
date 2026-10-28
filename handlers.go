package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync/atomic"
)

// /api/ endpoint handlers
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
	w.Write([]byte("OK"))
}

func censorChirp(body string) string {
	bodySlice := strings.Split(body, " ")
	bad_words := map[string]bool{"kerfuffle": true, "sharbert": true, "fornax": true}
	for i, word := range bodySlice {
		if bad_words[strings.ToLower(word)] {
			bodySlice[i] = "****"
		}
	}
	return strings.Join(bodySlice, " ")
}

func respondWithErrors(w http.ResponseWriter, statusCode int, msg string) {
	type errorResponse struct {
		Error string `json:"error"`
	}
	err := errorResponse{
		Error: msg,
	}
	respondWithPayload(w, statusCode, err)
}

func respondWithPayload(w http.ResponseWriter, statusCode int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	dat, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Error marshalling json %s\n", err)
		w.WriteHeader(500)
		return
	}
	w.WriteHeader(statusCode)
	w.Write(dat)
}

func validateChirpHandler(w http.ResponseWriter, r *http.Request) {
	type chirp struct {
		Body string `json:"body"`
	}

	type cleaned struct {
		Cleaned string `json:"cleaned_body"`
	}

	/*
		type valid struct {
			Valid bool `json:"valid"`
		}
	*/

	decoder := json.NewDecoder(r.Body)
	chirpToCheck := chirp{}

	err := decoder.Decode(&chirpToCheck)
	if err != nil {
		respondWithErrors(w, 400, "Bad Request")
		return
	}

	chirpLen := len(chirpToCheck.Body)
	if chirpLen > 140 {
		respondWithErrors(w, 400, "Chirp is too long")
		return
	}

	cleaned_body := censorChirp(chirpToCheck.Body)

	/*
		val := valid{
			Valid: true,
		}
	*/
	clean := cleaned{
		Cleaned: cleaned_body,
	}
	respondWithPayload(w, 200, clean)
	return

}

// API Config Related Handlers at /admin/ endpoints
type apiConfig struct {
	fileserverHits atomic.Int32
}

func (cfg *apiConfig) middlewareMetricsIncrement(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func (cfg *apiConfig) metricHandler(w http.ResponseWriter, r *http.Request) {
	hitsCount := cfg.fileserverHits.Load()
	hitsString := fmt.Sprintf("<html><body><h1>Welcome, Chirpy Admin</h1><p>Chirpy has been visited %d times!</p></body></html>", hitsCount)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(200)
	w.Write([]byte(hitsString))
}

func (cfg *apiConfig) resetHandler(w http.ResponseWriter, r *http.Request) {
	cfg.fileserverHits.Store(0)
}
