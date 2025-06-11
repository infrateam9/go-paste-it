package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
)

type server struct {
	router  *http.ServeMux
	store   store
	baseURL string
}

// setHTMLHeaders sets common headers for HTML responses
func setHTMLHeaders(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
}

func newServer(store store) (*server, error) {
	baseURL := os.Getenv("PASTE_BASE_URL")
	if baseURL != "" {
		log.Printf("Using custom base URL: %s", baseURL)
	}
	s := &server{
		store:   store,
		baseURL: baseURL,
	}
	s.init()
	return s, nil
}

func (s *server) init() {
	s.router = http.NewServeMux()
	s.router.HandleFunc("/_health", s.handleHealthCheck)
	s.router.HandleFunc("/", s.handleHomePage)
	s.router.HandleFunc("/paste", s.HandlePaste)
	s.router.HandleFunc("/view/", s.HandleView)

	// Add content type headers for static files
	fs := http.FileServer(http.Dir("static"))
	wrappedFs := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, ".css") {
			w.Header().Set("Content-Type", "text/css; charset=utf-8")
		}
		fs.ServeHTTP(w, r)
	})
	s.router.Handle("/static/", http.StripPrefix("/static/", wrappedFs))
}

func (s *server) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write([]byte(fmt.Sprintf("Application is healthy")))
}

func (s *server) handleHomePage(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	staticBaseURL := os.Getenv("S3_STATIC_BASE_URL")
	templates["index.html"].Execute(w, map[string]interface{}{
		"StaticBaseURL": staticBaseURL,
	})
}
