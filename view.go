package main

import (
	"log"
	"net/http"
	"os"
	"time"
)

func (s *server) HandleView(w http.ResponseWriter, r *http.Request) {
	// Set Content-Type header at the start of the handler
	setHTMLHeaders(w)
	staticBaseURL := os.Getenv("S3_STATIC_BASE_URL")

	id := r.URL.Path[len("/view/"):]
	snippet, err := s.store.GetSnippet(r.Context(), id)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusNotFound)
		templates["404.html"].Execute(w, map[string]interface{}{
			"StaticBaseURL": staticBaseURL,
		})
		return
	}

	if snippet.Expiration.Before(time.Now()) {
		// Delete expired snippet from DB
		_ = s.store.DeleteSnippet(r.Context(), id)
		http.Error(w, "Paste has expired", http.StatusGone)
		return
	}
	// If password protection is enabled
	if snippet.EnablePassword {
		password := r.FormValue("password")
		if password == "" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			templates["password.html"].Execute(w, map[string]interface{}{
				"ID":            id,
				"StaticBaseURL": staticBaseURL,
			})
			return
		}
		if !checkPasswordHash(password, snippet.Password) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			templates["password.html"].Execute(w, map[string]interface{}{
				"ID":            id,
				"ErrorMessage":  "Invalid password. Please try again.",
				"StaticBaseURL": staticBaseURL,
			})
			return
		}
	}

	log.Printf("[%s] Starting view process for ID=%s", time.Now().Format(time.RFC3339Nano), id)

	// For burn-after-read, handle it atomically to prevent race conditions
	if snippet.BurnAfterRead {
		log.Printf("[%s] Processing burn-after-read snippet ID=%s",
			time.Now().Format(time.RFC3339Nano), id)

		// Set headers first, before any potential errors
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, private")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")

		// First try to increment view count - this acts as our atomic check
		snippet.ViewCount++
		if err := s.store.UpdateSnippet(r.Context(), id, snippet); err != nil {
			if os.IsNotExist(err) {
				http.Error(w, "This paste has already been viewed and deleted", http.StatusGone)
				return
			}
			log.Printf("[%s] Failed to update view count: %v",
				time.Now().Format(time.RFC3339Nano), err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// If this is the first view (now count is 1), show content and delete
		if snippet.ViewCount == 1 {
			// Render the template before deletion to ensure the user sees the content
			log.Printf("[%s] Rendering final view for burn-after-read ID=%s",
				time.Now().Format(time.RFC3339Nano), id)
			if err := templates["view.html"].Execute(w, map[string]interface{}{
				"Created":       snippet.CreatedAt.Local().String(),
				"Content":       snippet.Content,
				"BurnAfterRead": "true",
				"StaticBaseURL": staticBaseURL,
			}); err != nil {
				log.Printf("[%s] Failed to render template: %v",
					time.Now().Format(time.RFC3339Nano), err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			// Delete after showing content
			if err := s.store.DeleteSnippet(r.Context(), id); err != nil && !os.IsNotExist(err) {
				log.Printf("[%s] Failed to delete burn-after-read snippet: %v",
					time.Now().Format(time.RFC3339Nano), err)
				// Don't return error to client since they already saw the content
			}
			return
		}

		// If we get here, it means someone else viewed it first
		http.Error(w, "This paste has already been viewed and deleted", http.StatusGone)
		return
	}

	// For non-burn-after-read snippets, increment view count
	log.Printf("[%s] Incrementing view count for normal snippet ID=%s (current count: %d)",
		time.Now().Format(time.RFC3339Nano), id, snippet.ViewCount)
	snippet.ViewCount++
	if err := s.store.UpdateSnippet(r.Context(), id, snippet); err != nil {
		log.Printf("[%s] Failed to update snippet view count: %v",
			time.Now().Format(time.RFC3339Nano), err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Render the template for normal snippets
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	log.Printf("[%s] Rendering template for normal snippet ID=%s",
		time.Now().Format(time.RFC3339Nano), id)
	if err := templates["view.html"].Execute(w, map[string]interface{}{
		"Created":       snippet.CreatedAt.Local().String(),
		"Content":       snippet.Content,
		"StaticBaseURL": staticBaseURL,
	}); err != nil {
		log.Printf("[%s] Failed to render template: %v",
			time.Now().Format(time.RFC3339Nano), err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

}
