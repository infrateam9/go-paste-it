package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
)

type server struct {
	router  *gin.Engine
	store   store
	baseURL string
}

func setHTMLHeaders(c *gin.Context) {
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Header("X-Content-Type-Options", "nosniff")
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
	r := gin.Default()

	r.GET("/_health", s.handleHealthCheck)
	r.GET("/", s.handleHomePage)
	r.POST("/paste", s.handlePaste)
	r.GET("/view/:id", s.handleView)
	r.POST("/view/:id", s.handleView) // for password-protected pastes

	// Static file serving with custom headers for CSS
	r.GET("/static/*filepath", func(c *gin.Context) {
		file := c.Param("filepath")
		if len(file) > 4 && file[len(file)-4:] == ".css" {
			c.Header("Content-Type", "text/css; charset=utf-8")
			c.Header("X-Content-Type-Options", "nosniff")
		}
		c.File("./static" + file)
	})

	s.router = r
}

func (s *server) handleHealthCheck(c *gin.Context) {
	c.Header("Content-Type", "text/plain; charset=utf-8")
	c.String(200, "Application is healthy")
}

func (s *server) handleHomePage(c *gin.Context) {
	setHTMLHeaders(c)
	err := templates["index.html"].Execute(c.Writer, nil)
	if err != nil {
		log.Printf("Failed to render index.html: %v", err)
		c.Status(500)
	}
}
