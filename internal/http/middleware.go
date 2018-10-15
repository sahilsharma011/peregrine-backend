package http

import (
	"log"
	"net/http"
)

// CORS is a middleware for setting Cross Origin Resource Sharing headers.
func CORS(next http.Handler, origin string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, PATCH, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")

		next.ServeHTTP(w, r)
	})
}

// LimitBody is middleware to protect the server from requests containing
// massive amounts of data.
func LimitBody(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 1000000) // 1 MB
		next.ServeHTTP(w, r)
	})
}

// Log logs information about incoming HTTP requests.
func Log(next http.Handler, l *log.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		l.Printf("Info: %s request from %s to %s", r.Method, r.RemoteAddr, r.URL.String())

		next.ServeHTTP(w, r)
	})
}
