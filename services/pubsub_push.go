package services

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/url"

	"github.com/sebasegovia01/base-template-go-gin/config"
)

type PubSubService struct {
	cfg *config.Config
}

type PubSubMessage struct {
	DeliveryAttempt *int `json:"deliveryAttempt,omitempty"`
	Message         struct {
		Attributes   map[string]string `json:"attributes,omitempty"`
		Data         string            `json:"data"`
		ID           string            `json:"messageId"`
		MessageID    string            `json:"message_id,omitempty"`
		OrderingKey  *string           `json:"orderingKey,omitempty"`
		PublishTime  string            `json:"publishTime"`
		PublishTime2 string            `json:"publish_time,omitempty"`
	} `json:"message"`
	Subscription string `json:"subscription"`
}

type StorageEvent struct {
	Bucket string `json:"bucket"`
	Name   string `json:"name"`
}

func NewPubSubService(cfg *config.Config) (*PubSubService, error) {
	log.Printf("Initializing PubSub push service for project: %s", cfg.GCPCredentials.ProjectID)
	return &PubSubService{
		cfg: cfg,
	}, nil
}

func (s *PubSubService) ExtractStorageEvent(body io.ReadCloser) (*StorageEvent, error) {
	bodyBytes, err := io.ReadAll(body)
	if err != nil {
		return nil, fmt.Errorf("error reading request body: %w", err)
	}

	var message PubSubMessage
	if err := json.Unmarshal(bodyBytes, &message); err != nil {
		return nil, fmt.Errorf("error unmarshalling message: %w", err)
	}

	log.Printf("Received message: ID=%s, PublishTime=%s", message.Message.ID, message.Message.PublishTime)
	if message.DeliveryAttempt != nil {
		log.Printf("Delivery attempt: %d", *message.DeliveryAttempt)
	}
	if len(message.Message.Attributes) > 0 {
		log.Printf("Message attributes: %v", message.Message.Attributes)
	}

	// Decode the base64 encoded data
	decodedData, err := base64.StdEncoding.DecodeString(message.Message.Data)
	if err != nil {
		return nil, fmt.Errorf("error decoding message data: %w", err)
	}

	var storageEvent StorageEvent
	if err := json.Unmarshal(decodedData, &storageEvent); err != nil {
		return nil, fmt.Errorf("error unmarshalling storage event data: %w", err)
	}

	// Decodificar el nombre del objeto
	decodedName, err := url.QueryUnescape(storageEvent.Name)
	if err != nil {
		return nil, fmt.Errorf("error decoding object name: %w", err)
	}
	storageEvent.Name = decodedName

	if storageEvent.Bucket == "" {
		return nil, fmt.Errorf("bucket name not found in message")
	}

	if storageEvent.Name == "" {
		return nil, fmt.Errorf("object name not found in message")
	}

	log.Printf("Extracted storage event: Bucket=%s, Name=%s", storageEvent.Bucket, storageEvent.Name)

	return &storageEvent, nil
}
