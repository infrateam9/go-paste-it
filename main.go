package main

import (
	"fmt"
	"html/template"
	"log"
	"os"
	"path/filepath"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/awslabs/aws-lambda-go-api-proxy/httpadapter"
)

var templates = make(map[string]*template.Template)

func loadTemplates() error {
	templateDir := "templates"
	pattern := filepath.Join(templateDir, "*.html")

	files, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("error finding templates: %v", err)
	}
	for _, file := range files {
		name := filepath.Base(file)
		tmpl, err := template.ParseFiles(file)
		if err != nil {
			return fmt.Errorf("error parsing template %s: %v", name, err)
		}
		templates[name] = tmpl
	}
	return nil
}

func lambdaHandler() {
	if err := loadTemplates(); err != nil {
		log.Fatalf("Error loading templates: %v", err)
	}

	tableName := os.Getenv("PASTE_DYNAMO_TABLE")
	if tableName == "" {
		tableName = "go-paste-it-snippets"
	}
	store, err := newDynamoStore(tableName)
	if err != nil {
		log.Fatalf("FATAL: Cannot connect to DynamoDB: %v", err)
	}
	s, err := newServer(store)
	if err != nil {
		log.Fatalf("Error creating server: %v", err)
	}

	adapter := httpadapter.NewV2(s.router)
	lambda.Start(adapter.ProxyWithContext)
}

func main() {
	lambdaHandler()
}
