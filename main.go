package main

import _ "github.com/lib/pq"
import "github.com/joho/godotenv"
import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"sync/atomic"
	"encoding/json"
	"time"
	"strings"
	"slices"
	"os"
	"github.com/skookum-ua/http-server/internal/database"
)

type apiConfig struct {
fileserverHits atomic.Int32
dbQueries *database.Queries
}

func censore(chirp string) string{
	veryBadWords:= []string{"kerfuffle", "sharbert", "fornax"}
	separated:= strings.Split(chirp, " ")
	for i, badword := range separated{
		if slices.Contains(veryBadWords, strings.ToLower(badword)){
			separated[i] = "****"
		}
	}
	return strings.Join(separated, " ")
}

func handlerHealth(w http.ResponseWriter,h *http.Request){
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
	w.Write([]byte("OK"))
}

func (c *apiConfig) handlerHits(w http.ResponseWriter,h *http.Request){
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(200)
	st := fmt.Sprintf("<html><body><h1>Welcome, Chirpy Admin</h1><p>Chirpy has been visited %d times!</p></body></html>", c.fileserverHits.Load())
	w.Write([]byte(st))
}

func (c *apiConfig) handlerReset(w http.ResponseWriter,h *http.Request){
	c.fileserverHits.Swap(0)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
	w.Write([]byte("Hits reseted"))
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s", r.Method, r.URL.Path)
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func respondeWithError(w http.ResponseWriter, code int, msg string){
	type returnVals struct {
        CreatedAt time.Time `json:"created_at"`
		Code int `json:"code"`
		Error string `json:"error"`
	}
	  respBody := returnVals{
        CreatedAt: time.Now(),
        Code: code,
		Error: msg,
    }
	respondWithJSON(w, code, respBody)
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}){
    dat, err := json.Marshal(payload)
	if err != nil {
			log.Printf("Error marshalling JSON: %s", err)
			w.WriteHeader(500)
			return
	}
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(code)
    w.Write(dat)
}

func handlerValidate(w http.ResponseWriter, r *http.Request){
    type parameters struct {
        Body string `json:"body"`
    }
	type response struct {
        Body string `json:"cleaned_body"`
    }

    decoder := json.NewDecoder(r.Body)
    params := parameters{}
    err := decoder.Decode(&params)
    if err != nil {
		log.Printf("Error decoding parameters: %s", err)
		w.WriteHeader(500)
		respondeWithError(w, 500, "Error decoding parameters")
		return
    }
    if len(params.Body) > 140{
		respondeWithError(w, 400, "Chirp is too long")
		return
	}
	res:=response{}
	res.Body = censore(params.Body)
	respondWithJSON(w, 200, res)
}

func main(){
	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	db, err := sql.Open("postgres", dbURL)
	if err != nil{
		fmt.Printf("Error opening database: %s", err)
	}
	dbQueries := database.New(db)

	var apiCfg apiConfig
	apiCfg.dbQueries = dbQueries

	mux := http.NewServeMux()
	mux.Handle("/app/",  http.StripPrefix("/app", apiCfg.middlewareMetricsInc(http.FileServer(http.Dir(".")))))
	mux.HandleFunc("GET /api/healthz" , handlerHealth)
	mux.HandleFunc("GET /admin/metrics" , apiCfg.handlerHits)
	mux.HandleFunc("POST /admin/reset" , apiCfg.handlerReset)
	mux.HandleFunc("POST /api/validate_chirp" , handlerValidate)
	
	server := &http.Server{}
	server.Addr = ":8080"
	server.Handler = mux

	server.ListenAndServe()
}
