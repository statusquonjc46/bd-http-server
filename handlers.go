package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	//"uuid"

	"github.com/google/uuid"
	"github.com/statusquonjc46/bd-http-server/internal/database"
)

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
	w.Write([]byte("OK"))
}

// HELPER FUNCTIONS
// helper function for censoring chirps
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

// helper function for creating error http responses
func respondWithErrors(w http.ResponseWriter, statusCode int, msg string) {
	type errorResponse struct {
		Error string `json:"error"`
	}
	err := errorResponse{
		Error: msg,
	}
	respondWithPayload(w, statusCode, err)
}

// helper function for creating json reponse payloads
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

// API FUNCTIONS
// api function to handle validation and insertion of a chirp to chirps table
func (cfg *apiConfig) chirpHandler(w http.ResponseWriter, r *http.Request) {
	type reqChirp struct {
		Body    string        `json:"body"`
		User_Id uuid.NullUUID `json:"user_id"`
	}

	type cleaned struct {
		Cleaned string `json:"cleaned_body"`
	}

	decoder := json.NewDecoder(r.Body)
	chirpToCheck := reqChirp{}

	err := decoder.Decode(&chirpToCheck)
	if err != nil {
		log.Println(err)
		respondWithErrors(w, 400, "Bad Request")
		return
	}

	chirpLen := len(chirpToCheck.Body)
	if chirpLen > 140 {
		log.Println(err)
		respondWithErrors(w, 400, "Chirp is too long")
		return
	}

	cleaned_body := censorChirp(chirpToCheck.Body)

	argChirp := database.CreateChirpParams{
		Body:   cleaned_body,
		UserID: chirpToCheck.User_Id,
	}

	createdChirp, err := cfg.database.CreateChirp(r.Context(), argChirp)
	if err != nil {
		log.Println(err)
		respondWithErrors(w, 500, "Failed to create chirp in database")
		return
	}
	payloadChirp := Chirp{
		ID:        createdChirp.ID,
		CreatedAt: createdChirp.CreatedAt.Time,
		UpdatedAt: createdChirp.UpdatedAt.Time,
		Body:      createdChirp.Body,
		UserID:    createdChirp.UserID,
	}
	respondWithPayload(w, 201, payloadChirp)
}

// api function to get all chirps
func (cfg *apiConfig) getAllChirpsHandler(w http.ResponseWriter, r *http.Request) {
	allChirps, err := cfg.database.GetAllChirps(r.Context())
	chirpSlice := []Chirp{}

	if err != nil {
		log.Println(err)
		respondWithErrors(w, 500, "Failed to query database for all chirps")
		return
	}
	for _, chirp := range allChirps {
		appendChirp := Chirp{
			ID:        chirp.ID,
			CreatedAt: chirp.CreatedAt.Time,
			UpdatedAt: chirp.UpdatedAt.Time,
			Body:      chirp.Body,
			UserID:    chirp.UserID,
		}

		chirpSlice = append(chirpSlice, appendChirp)
	}
	respondWithPayload(w, 200, chirpSlice)
}

func (cfg *apiConfig) getAChirp(w http.ResponseWriter, r *http.Request) {
	type chirpId struct {
		ID uuid.UUID
	}
	pathVal := r.PathValue("chirpId")
	parsedVal, err := uuid.Parse(pathVal)
	if err != nil {
		log.Println(err)
		respondWithErrors(w, 400, "Bad request")
		return
	}

	returnedChirp, err := cfg.database.GetAChirp(r.Context(), parsedVal)
	if err != nil {
		log.Println(err)
		respondWithErrors(w, 404, "Failed to query database for chirp")
		return
	}

	payloadChirp := Chirp{
		ID:        returnedChirp.ID,
		CreatedAt: returnedChirp.CreatedAt.Time,
		UpdatedAt: returnedChirp.UpdatedAt.Time,
		Body:      returnedChirp.Body,
		UserID:    returnedChirp.UserID,
	}

	respondWithPayload(w, 200, payloadChirp)

}

// api function to handle adding a user to users table
func (cfg *apiConfig) addUserHandler(w http.ResponseWriter, r *http.Request) {
	type reqEmail struct {
		Email string `json:"email"`
	}

	decoder := json.NewDecoder(r.Body)
	emailBody := reqEmail{}

	err := decoder.Decode(&emailBody)
	if err != nil {
		log.Println(err)
		respondWithErrors(w, 400, "Bad Request")
		return
	}

	user, err := cfg.database.CreateUser(r.Context(), emailBody.Email)
	if err != nil {
		log.Println(err)
		respondWithErrors(w, 500, "Failed to add user")
		return
	}
	jsonUser := User{
		ID:        uuid.UUID(user.ID),
		CreatedAt: user.CreatedAt.Time,
		UpdatedAt: user.UpdatedAt.Time,
		Email:     user.Email,
	}
	respondWithPayload(w, 201, jsonUser)
}

// api middleware to count page visits
func (cfg *apiConfig) middlewareMetricsIncrement(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

// api function to return the visit count
func (cfg *apiConfig) metricHandler(w http.ResponseWriter, r *http.Request) {
	hitsCount := cfg.fileserverHits.Load()
	hitsString := fmt.Sprintf("<html><body><h1>Welcome, Chirpy Admin</h1><p>Chirpy has been visited %d times!</p></body></html>", hitsCount)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(200)
	w.Write([]byte(hitsString))
}

// api function to reset the count of visits and reset the users table
func (cfg *apiConfig) resetHandler(w http.ResponseWriter, r *http.Request) {
	if cfg.platform != "dev" {
		respondWithErrors(w, 403, "Forbidden")
	}
	err := cfg.database.DeleteAllUsers(r.Context())
	if err != nil {
		log.Println(err)
		respondWithErrors(w, 500, "Failed to reset table")
		return
	}
	cfg.fileserverHits.Store(0)
}
