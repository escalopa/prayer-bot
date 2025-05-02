package internal

import (
	"encoding/json"
	"fmt"

	"github.com/go-telegram/bot/models"
)

type RequestBody struct {
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body"`
}

func ParseRequest(request []byte) (*models.Update, map[string]string, error) {
	requestBody := &RequestBody{}

	if err := json.Unmarshal(request, &requestBody); err != nil {
		return nil, nil, fmt.Errorf("unmarshal request body: %v", err)
	}

	update := &models.Update{}
	if err := json.Unmarshal([]byte(requestBody.Body), update); err != nil {
		return nil, nil, fmt.Errorf("unmarshal update: %v", err)
	}

	return update, requestBody.Headers, nil
}
