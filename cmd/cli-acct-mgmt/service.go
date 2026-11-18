package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/AndrewSerra/kitchen/internal/models"
	"github.com/AndrewSerra/kitchen/internal/utils"
)

type ServiceName string

const (
	AuthServiceName   ServiceName = "auth"
	StreamServiceName ServiceName = "streams"
)

const REQUEST_TIMEOUT = 10 * time.Second

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
			string(AuthServiceName):   "auth",
			string(StreamServiceName): "streams",
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

func (apiService *APIService) addJWTToRequest(req *http.Request, token string) error {
	if token == "" {
		return fmt.Errorf("token is required")
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	return nil
}

func (apiService *APIService) SendRegisterRequest(email string, password string) (*models.RegisterResponse, error) {
	registerRequest := models.RegisterRequest{
		Email:    email,
		Password: password,
	}
	if err := utils.ValidateRegisterRequest(&registerRequest); err != nil {
		return nil, err
	}

	requestURL := apiService.getAPIURL(string(AuthServiceName), "register")

	reqBody, err := json.Marshal(registerRequest)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), REQUEST_TIMEOUT)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, requestURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	response, err := apiService.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("failed to register: %s", response.Status)
	}

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	var registerResponse models.RegisterResponse
	if err := json.Unmarshal(responseBody, &registerResponse); err != nil {
		return nil, err
	}

	if !registerResponse.Success {
		return nil, fmt.Errorf("failed to register: %s", registerResponse.Message)
	}

	return &registerResponse, nil
}

func (apiService *APIService) SendLogoutRequest(refreshToken string) (*models.LogoutResponse, error) {
	requestURL := apiService.getAPIURL(string(AuthServiceName), "logout")

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, requestURL, nil)
	if err != nil {
		return nil, err
	}
	if err := apiService.addJWTToRequest(req, refreshToken); err != nil {
		return nil, err
	}
	response, err := apiService.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to logout: %s", response.Status)
	}

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	var logoutResponse models.LogoutResponse
	if err := json.Unmarshal(responseBody, &logoutResponse); err != nil {
		return nil, err
	}

	return &logoutResponse, nil
}

func (apiService *APIService) SendCreateStreamRequest(name string, accessToken string) (*models.Stream, error) {
	requestURL := apiService.getAPIURL(string(StreamServiceName), "")
	createStreamRequest := models.CreateStreamRequest{
		Name: name,
	}

	if err := utils.ValidateCreateStreamRequest(&createStreamRequest); err != nil {
		return nil, err
	}

	reqBody, err := json.Marshal(createStreamRequest)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), REQUEST_TIMEOUT)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, requestURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if err := apiService.addJWTToRequest(req, accessToken); err != nil {
		return nil, err
	}
	response, err := apiService.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("failed to create stream: %s", response.Status)
	}

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	var createStreamResponse models.CreateStreamResponse
	if err := json.Unmarshal(responseBody, &createStreamResponse); err != nil {
		return nil, err
	}

	if !createStreamResponse.Success {
		return nil, fmt.Errorf("failed to create stream: %s", createStreamResponse.Message)
	}

	return createStreamResponse.Stream, nil
}

func (apiService *APIService) SendGetStreamByNameRequest(name string, accessToken string) (*models.Stream, error) {
	urlValues := url.Values{}
	urlValues.Set("name", name)
	requestURL := apiService.getAPIURL(string(StreamServiceName), "?"+urlValues.Encode())

	ctx, cancel := context.WithTimeout(context.Background(), REQUEST_TIMEOUT)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, err
	}
	if err := apiService.addJWTToRequest(req, accessToken); err != nil {
		return nil, err
	}
	response, err := apiService.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get stream: %s", response.Status)
	}

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	var getStreamResponse models.GetAllStreamsResponse
	if err := json.Unmarshal(responseBody, &getStreamResponse); err != nil {
		return nil, err
	}

	if !getStreamResponse.Success {
		return nil, fmt.Errorf("failed to get stream: %s", getStreamResponse.Message)
	}

	if len(getStreamResponse.Streams) == 0 {
		return nil, fmt.Errorf("stream not found")
	}

	return getStreamResponse.Streams[0], nil
}

func (apiService *APIService) SendGetAllStreamsRequest(accessToken string) (*models.Streams, error) {
	requestURL := apiService.getAPIURL(string(StreamServiceName), "")

	ctx, cancel := context.WithTimeout(context.Background(), REQUEST_TIMEOUT)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, err
	}
	if err := apiService.addJWTToRequest(req, accessToken); err != nil {
		return nil, err
	}
	response, err := apiService.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get all streams: %s", response.Status)
	}

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	var getAllStreamsResponse models.GetAllStreamsResponse
	if err := json.Unmarshal(responseBody, &getAllStreamsResponse); err != nil {
		return nil, err
	}

	if !getAllStreamsResponse.Success {
		return nil, fmt.Errorf("failed to get all streams: %s", getAllStreamsResponse.Message)
	}

	return &getAllStreamsResponse.Streams, nil
}

func (apiService *APIService) SendDeleteStreamRequest(name string, accessToken string) error {
	streams, err := apiService.SendGetStreamByNameRequest(name, accessToken)
	if err != nil {
		return err
	}

	requestURL := apiService.getAPIURL(string(StreamServiceName), fmt.Sprintf("%d", streams.ID))

	ctx, cancel := context.WithTimeout(context.Background(), REQUEST_TIMEOUT)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, requestURL, nil)
	if err != nil {
		return err
	}
	if err := apiService.addJWTToRequest(req, accessToken); err != nil {
		return err
	}
	response, err := apiService.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to delete stream: %s", response.Status)
	}

	return nil
}
