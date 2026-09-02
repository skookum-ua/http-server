package main

import _ "github.com/lib/pq"
import "github.com/joho/godotenv"
import (
	"database/sql"
	"fmt"

	"net/http"
	"sync/atomic"

	"time"
	"os"
	"github.com/skookum-ua/http-server/internal/database"
	"github.com/google/uuid"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	dbQueries *database.Queries
	platform string
}

type User struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
}

type Chirp struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Body     string    `json:"body"`
	UserID uuid.UUID	`json:"user_id"`
}


func main(){
	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	platf := os.Getenv("PLATFORM")
	db, err := sql.Open("postgres", dbURL)
	if err != nil{
		fmt.Printf("Error opening database: %s", err)
	}
	dbQueries := database.New(db)

	var apiCfg apiConfig
	apiCfg.dbQueries = dbQueries
	apiCfg.platform = platf

	mux := http.NewServeMux()
	mux.Handle("/app/",  http.StripPrefix("/app", apiCfg.middlewareMetricsInc(http.FileServer(http.Dir(".")))))
	mux.HandleFunc("GET /api/healthz" , handlerHealth)
	mux.HandleFunc("GET /admin/metrics" , apiCfg.handlerHits)
	mux.HandleFunc("POST /admin/reset" , apiCfg.handlerReset)
	mux.HandleFunc("POST /api/validate_chirp" , handlerValidate)
	mux.HandleFunc("POST /api/users" , apiCfg.handlerCreateUsers)
	mux.HandleFunc("POST /api/chirps" , apiCfg.handlerChirps)
	mux.HandleFunc("GET /api/chirps" , apiCfg.handlerAllChirps)
	mux.HandleFunc("GET /api/chirps/{chirpID}", apiCfg.handlerGetChirpsId)
	
	server := &http.Server{}
	server.Addr = ":8080"
	server.Handler = mux

	server.ListenAndServe()
}
