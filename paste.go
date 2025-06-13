package main

import (
	"log"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

const maxSnippetSize = 64 * 1024 // 64 KB

func (s *server) handlePaste(c *gin.Context) {
	setHTMLHeaders(c)

	if c.Request.Method != "POST" {
		c.String(405, "Method not allowed")
		return
	}

	if err := c.Request.ParseForm(); err != nil {
		c.String(400, "Invalid form data")
		return
	}

	content := c.PostForm("content")
	if len(content) == 0 {
		c.String(400, "Content cannot be empty")
		return
	}
	if len(content) > maxSnippetSize {
		c.String(413, "Content is too large")
		return
	}
	title := c.PostForm("title")
	expiration := c.PostForm("expiration")
	burnAfterRead := c.PostForm("burn_after_read") == "on"
	enablePassword := c.PostForm("enable_password") == "on"
	password := c.PostForm("password")

	var hashedPassword string
	if enablePassword {
		var err error
		hashedPassword, err = hashPassword(password)
		if err != nil {
			log.Println(err)
			c.String(500, "Internal server error")
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

	if err := s.store.PutSnippet(c.Request.Context(), id, &snippet); err != nil {
		log.Println(err)
		c.String(500, "Internal server error")
		return
	}

	var viewURL string
	if s.baseURL != "" {
		viewURL = s.baseURL + "/view/" + id
	} else {
		scheme := "http"
		if c.Request.TLS != nil {
			scheme = "https"
		}
		viewURL = scheme + "://" + c.Request.Host + "/view/" + id
	}

	staticBaseURL := os.Getenv("S3_STATIC_BASE_URL")
	if err := templates["created.html"].Execute(c.Writer, map[string]interface{}{
		"Title":          title,
		"URL":            viewURL,
		"BurnAfterRead":  burnAfterRead,
		"EnablePassword": enablePassword,
		"StaticBaseURL":  staticBaseURL,
	}); err != nil {
		log.Printf("Failed to execute created template: %v", err)
		c.String(500, "Internal Server Error")
		return
	}
}
