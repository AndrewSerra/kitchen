package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/AndrewSerra/kitchen/internal/models"
	"github.com/AndrewSerra/kitchen/internal/utils"
)

type ServiceName string

const (
	AuthServiceName ServiceName = "auth"
)

type APIService struct {
	serverURL   string
	pathPrefix  string
	servicePath map[string]string
	httpClient  *http.Client
}

func NewAPIService(serverURL string) *APIService {
	return &APIService{
		serverURL:  serverURL,
		pathPrefix: "api",
		servicePath: map[string]string{
			string(AuthServiceName): "auth",
		},
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (apiService *APIService) getAPIURL(serviceName string, path string) string {
	apiService.serverURL = strings.TrimSuffix(apiService.serverURL, "/")
	basePath := fmt.Sprintf("%s/%s", apiService.serverURL, apiService.pathPrefix)
	return fmt.Sprintf("%s/%s/%s", basePath, apiService.servicePath[serviceName], path)
}

func (apiService *APIService) SendRegisterRequest(email string, password string) error {
	registerRequest := models.RegisterRequest{
		Email:    email,
		Password: password,
	}
	if err := utils.ValidateRegisterRequest(&registerRequest); err != nil {
		return err
	}

	url := apiService.getAPIURL(string(AuthServiceName), "register")

	reqBody, err := json.Marshal(registerRequest)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(reqBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	response, err := apiService.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to register: %s", response.Status)
	}

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return err
	}

	var registerResponse models.RegisterResponse
	if err := json.Unmarshal(responseBody, &registerResponse); err != nil {
		return err
	}

	if !registerResponse.Success {
		return fmt.Errorf("failed to register: %s", registerResponse.Message)
	}

	fmt.Println("Registered successfully")
	fmt.Println("User ID:", registerResponse.User.ID)
	fmt.Println("User Email:", registerResponse.User.Email)
	fmt.Println("User Created At:", registerResponse.User.CreatedAt)
	fmt.Println("User Updated At:", registerResponse.User.UpdatedAt)

	return nil
}
