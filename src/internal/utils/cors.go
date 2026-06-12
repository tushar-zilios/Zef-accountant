package utils

import (
	"net/http"
)

// CORSMiddleware handles Cross-Origin Resource Sharing (CORS) headers.
// It allows all origins (*), standard HTTP methods, and common headers like Content-Type and Authorization.
// It also responds immediately to preflight OPTIONS requests with a 200 OK status.
func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, ngrok-skip-browser-warning, x-workspace-id")
		
		// If it's a preflight request, respond immediately with 200 OK
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}
