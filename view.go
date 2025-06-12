package main

import (
	"log"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

func (s *server) handleView(c *gin.Context) {
	setHTMLHeaders(c)
	staticBaseURL := os.Getenv("S3_STATIC_BASE_URL")

	id := c.Param("id")
	snippet, err := s.store.GetSnippet(c.Request.Context(), id)
	if err != nil {
		log.Println(err)
		c.Status(404)
		templates["404.html"].Execute(c.Writer, map[string]interface{}{
			"StaticBaseURL": staticBaseURL,
		})
		return
	}

	if snippet.Expiration.Before(time.Now()) {
		_ = s.store.DeleteSnippet(c.Request.Context(), id)
		c.String(410, "Paste has expired")
		return
	}

	if snippet.EnablePassword {
		password := c.PostForm("password")
		if password == "" {
			setHTMLHeaders(c)
			templates["password.html"].Execute(c.Writer, map[string]interface{}{
				"ID":            id,
				"StaticBaseURL": staticBaseURL,
			})
			return
		}
		if !checkPasswordHash(password, snippet.Password) {
			setHTMLHeaders(c)
			templates["password.html"].Execute(c.Writer, map[string]interface{}{
				"ID":            id,
				"ErrorMessage":  "Invalid password. Please try again.",
				"StaticBaseURL": staticBaseURL,
			})
			return
		}
	}

	log.Printf("[%s] Starting view process for ID=%s", time.Now().Format(time.RFC3339Nano), id)

	if snippet.BurnAfterRead {
		log.Printf("[%s] Processing burn-after-read snippet ID=%s", time.Now().Format(time.RFC3339Nano), id)
		c.Header("Cache-Control", "no-store, no-cache, must-revalidate, private")
		c.Header("Pragma", "no-cache")
		c.Header("Expires", "0")
		snippet.ViewCount++
		if err := s.store.UpdateSnippet(c.Request.Context(), id, snippet); err != nil {
			if os.IsNotExist(err) {
				c.String(410, "This paste has already been viewed and deleted")
				return
			}
			log.Printf("[%s] Failed to update view count: %v", time.Now().Format(time.RFC3339Nano), err)
			c.String(500, "Internal Server Error")
			return
		}
		if snippet.ViewCount == 1 {
			log.Printf("[%s] Rendering final view for burn-after-read ID=%s", time.Now().Format(time.RFC3339Nano), id)
			if err := templates["view.html"].Execute(c.Writer, map[string]interface{}{
				"Created":       snippet.CreatedAt.Local().String(),
				"Content":       snippet.Content,
				"BurnAfterRead": "true",
				"StaticBaseURL": staticBaseURL,
			}); err != nil {
				log.Printf("[%s] Failed to render template: %v", time.Now().Format(time.RFC3339Nano), err)
				c.String(500, "Internal Server Error")
				return
			}
			if err := s.store.DeleteSnippet(c.Request.Context(), id); err != nil && !os.IsNotExist(err) {
				log.Printf("[%s] Failed to delete burn-after-read snippet: %v", time.Now().Format(time.RFC3339Nano), err)
			}
			return
		}
		c.String(410, "This paste has already been viewed and deleted")
		return
	}

	log.Printf("[%s] Incrementing view count for normal snippet ID=%s (current count: %d)", time.Now().Format(time.RFC3339Nano), id, snippet.ViewCount)
	snippet.ViewCount++
	if err := s.store.UpdateSnippet(c.Request.Context(), id, snippet); err != nil {
		log.Printf("[%s] Failed to update snippet view count: %v", time.Now().Format(time.RFC3339Nano), err)
		c.String(500, "Internal Server Error")
		return
	}

	log.Printf("[%s] Rendering template for normal snippet ID=%s", time.Now().Format(time.RFC3339Nano), id)
	if err := templates["view.html"].Execute(c.Writer, map[string]interface{}{
		"Created":       snippet.CreatedAt.Local().String(),
		"Content":       snippet.Content,
		"StaticBaseURL": staticBaseURL,
	}); err != nil {
		log.Printf("[%s] Failed to render template: %v", time.Now().Format(time.RFC3339Nano), err)
		c.String(500, "Internal Server Error")
		return
	}
}
