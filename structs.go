package main

import (
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/statusquonjc46/bd-http-server/internal/database"
)

const accessExpiresAt = time.Duration(3600) * time.Second

// user table entry
type User struct {
	ID          uuid.UUID `json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Email       string    `json:"email"`
	IsChirpyRed bool      `json:"is_chirpy_red"`
}

// user table with password hash entry
type UserWithHash struct {
	ID             uuid.UUID `json:"id"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	Email          string    `json:"email"`
	HashedPassword string    `json:"password"`
}

// user table with token entry
type UserWithToken struct {
	ID          uuid.UUID `json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Email       string    `json:"email"`
	Token       string    `json:"token"`
	Refresh     string    `json:"refresh_token"`
	IsChirpyRed bool      `json:"is_chirpy_red"`
}

// chirp table entry
type Chirp struct {
	ID        uuid.UUID     `json:"id"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
	Body      string        `json:"body"`
	UserID    uuid.NullUUID `json:"user_id"`
}

// api configuration settings
type apiConfig struct {
	fileserverHits atomic.Int32
	database       *database.Queries
	platform       string
	bearer         string
}
