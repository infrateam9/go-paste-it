package main

import (
	"log"
	"net/http"
	"os"
	"time"
)

const maxSnippetSize = 64 * 1024 // 64 KB

func (s *server) HandlePaste(w http.ResponseWriter, r *http.Request) {
	// Set Content-Type header at the start
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	content := r.FormValue("content")
	if len(content) == 0 {
		http.Error(w, "Content cannot be empty", http.StatusBadRequest)
		return
	}
	if len(content) > maxSnippetSize {
		http.Error(w, "Content is too large", http.StatusRequestEntityTooLarge)
		return
	}

	title := r.FormValue("title")
	expiration := r.FormValue("expiration")
	burnAfterRead := r.FormValue("burn_after_read") == "on"
	enablePassword := r.FormValue("enable_password") == "on"
	password := r.FormValue("password")

	var hashedPassword string
	if enablePassword {
		var err error
		hashedPassword, err = hashPassword(password)
		if err != nil {
			log.Println(err)
			http.Error(w, "Internal server Error", http.StatusInternalServerError)
			return
		}
	}

	id := generateID([]byte(content))

	snippet := Snippet{
		ID:             id,
		Title:          title,
		Expiration:     getExpirationTime(expiration),
		BurnAfterRead:  burnAfterRead,
		EnablePassword: enablePassword,
		Content:        content,
		Password:       hashedPassword,
		CreatedAt:      time.Now(),
	}

	if err := s.store.PutSnippet(r.Context(), id, &snippet); err != nil {
		log.Println(err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Use the server's baseURL for generating view URLs
	var viewURL string
	if s.baseURL != "" {
		viewURL = s.baseURL + "/view/" + id
	} else {
		scheme := "http"
		if r.TLS != nil {
			scheme = "https"
		}
		viewURL = scheme + "://" + r.Host + "/view/" + id
	}

	staticBaseURL := os.Getenv("S3_STATIC_BASE_URL")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := templates["created.html"].Execute(w, map[string]interface{}{
		"Title":          title,
		"URL":            viewURL,
		"BurnAfterRead":  burnAfterRead,
		"EnablePassword": enablePassword,
		"StaticBaseURL":  staticBaseURL,
	}); err != nil {
		log.Printf("Failed to execute created template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}
