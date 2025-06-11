package main

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type dynamoStore struct {
	db        *dynamodb.Client
	tableName string
}

func newDynamoStore(tableName string) (*dynamoStore, error) {
	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}
	db := dynamodb.NewFromConfig(cfg)
	return &dynamoStore{db: db, tableName: tableName}, nil
}

// Save a paste/snippet
func (s *dynamoStore) Save(ctx context.Context, id string, content string, password string, created time.Time) error {
	_, err := s.db.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(s.tableName),
		Item: map[string]types.AttributeValue{
			"id":       &types.AttributeValueMemberS{Value: id},
			"content":  &types.AttributeValueMemberS{Value: content},
			"password": &types.AttributeValueMemberS{Value: password},
			"created":  &types.AttributeValueMemberS{Value: created.Format(time.RFC3339)},
		},
	})
	return err
}

// Load a paste/snippet
func (s *dynamoStore) Load(ctx context.Context, id string) (content string, password string, created time.Time, err error) {
	res, err := s.db.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: id},
		},
	})
	if err != nil {
		return
	}
	item := res.Item
	if item == nil {
		err = fmt.Errorf("not found")
		return
	}
	content = item["content"].(*types.AttributeValueMemberS).Value
	password = item["password"].(*types.AttributeValueMemberS).Value
	created, err = time.Parse(time.RFC3339, item["created"].(*types.AttributeValueMemberS).Value)
	return
}

// Delete a paste/snippet
func (s *dynamoStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: id},
		},
	})
	return err
}

// List is not implemented for DynamoDB (scan is expensive)
func (s *dynamoStore) List(ctx context.Context) ([]string, error) {
	return nil, fmt.Errorf("list not implemented for DynamoDB store")
}

// dynamoStore implements the store interface
func (s *dynamoStore) PutSnippet(ctx context.Context, id string, snippet *Snippet) error {
	item := map[string]types.AttributeValue{
		"id":              &types.AttributeValueMemberS{Value: id},
		"title":           &types.AttributeValueMemberS{Value: snippet.Title},
		"expiration":      &types.AttributeValueMemberS{Value: snippet.Expiration.Format(time.RFC3339)},
		"burn_after_read": &types.AttributeValueMemberBOOL{Value: snippet.BurnAfterRead},
		"enable_password": &types.AttributeValueMemberBOOL{Value: snippet.EnablePassword},
		"password":        &types.AttributeValueMemberS{Value: snippet.Password},
		"content":         &types.AttributeValueMemberS{Value: snippet.Content},
		"view_count":      &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", snippet.ViewCount)},
		"created_at":      &types.AttributeValueMemberS{Value: snippet.CreatedAt.Format(time.RFC3339)},
	}
	_, err := s.db.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: &s.tableName,
		Item:      item,
	})
	return err
}

func (s *dynamoStore) GetSnippet(ctx context.Context, id string) (*Snippet, error) {
	res, err := s.db.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: &s.tableName,
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: id},
		},
	})
	if err != nil {
		return nil, err
	}
	item := res.Item
	if len(item) == 0 {
		return nil, fmt.Errorf("snippet not found")
	}
	var snippet Snippet
	if v, ok := item["title"].(*types.AttributeValueMemberS); ok {
		snippet.Title = v.Value
	}
	if v, ok := item["expiration"].(*types.AttributeValueMemberS); ok {
		t, _ := time.Parse(time.RFC3339, v.Value)
		snippet.Expiration = t
	}
	if v, ok := item["burn_after_read"].(*types.AttributeValueMemberBOOL); ok {
		snippet.BurnAfterRead = v.Value
	}
	if v, ok := item["enable_password"].(*types.AttributeValueMemberBOOL); ok {
		snippet.EnablePassword = v.Value
	}
	if v, ok := item["password"].(*types.AttributeValueMemberS); ok {
		snippet.Password = v.Value
	}
	if v, ok := item["content"].(*types.AttributeValueMemberS); ok {
		snippet.Content = v.Value
	}
	if v, ok := item["view_count"].(*types.AttributeValueMemberN); ok {
		fmt.Sscanf(v.Value, "%d", &snippet.ViewCount)
	}
	if v, ok := item["created_at"].(*types.AttributeValueMemberS); ok {
		t, _ := time.Parse(time.RFC3339, v.Value)
		snippet.CreatedAt = t
	}
	snippet.ID = id
	return &snippet, nil
}

func (s *dynamoStore) DeleteSnippet(ctx context.Context, id string) error {
	_, err := s.db.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: &s.tableName,
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: id},
		},
	})
	return err
}

func (s *dynamoStore) UpdateSnippet(ctx context.Context, id string, snippet *Snippet) error {
	return s.PutSnippet(ctx, id, snippet)
}
