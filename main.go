package main

import (
	"encoding/json"
	"log"
	"net/http"
)

const maxBodySize = 10 << 20 // 10 MB

// DiffRequest is the JSON body accepted by both diff endpoints.
type DiffRequest struct {
	Left  string `json:"left"`
	Right string `json:"right"`
}

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", handleHealth)
	mux.HandleFunc("POST /api/diff/text", handleTextDiff)
	mux.HandleFunc("POST /api/diff/json", handleJSONDiff)
	mux.HandleFunc("/", handleIndex)

	log.Println("Server listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", corsMiddleware(mux)))
}

// corsMiddleware adds permissive CORS headers so the API can be called from
// any origin (e.g. a separately hosted frontend or curl/fetch).
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("writeJSON encode error: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// handleIndex serves index.html for requests to "/".
func handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	http.ServeFile(w, r, "index.html")
}

// handleHealth responds with {"status":"ok"}.
func handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// handleTextDiff handles POST /api/diff/text.
//
// Request body (JSON):
//
//	{ "left": "<original text>", "right": "<comparison text>" }
//
// Response body: TextDiffResult JSON.
func handleTextDiff(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)

	var req DiffRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	result, err := TextDiff(req.Left, req.Right)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// handleJSONDiff handles POST /api/diff/json.
//
// Request body (JSON):
//
//	{ "left": "<JSON string>", "right": "<JSON string>" }
//
// Response body: JSONDiffResult JSON.
func handleJSONDiff(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)

	var req DiffRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	result, err := JSONDiff(req.Left, req.Right)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}
