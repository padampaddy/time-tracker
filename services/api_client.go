package services

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
)

type ApiClient struct {
	BaseURL string
	Token   string
}

func NewApiClient(baseURL string) *ApiClient {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		println("Unable to determine user home directory:", err)
		return &ApiClient{}
	}
	tokenPath := filepath.Join(homeDir, ".time-tracker", ".token")
	token := ""
	if data, err := os.ReadFile(tokenPath); err == nil {
		token = string(data)
	} else {
		println("Token file not found. Please login again.")
	}

	return &ApiClient{
		BaseURL: baseURL,
		Token:   token,
	}
}

func (c *ApiClient) Login(payload map[string]interface{}) (map[string]interface{}, error) {
	response, err := c.CallAPI("/api/login", "POST", payload)
	if err != nil {
		return nil, err
	}

	if token, ok := response["token"].(string); ok {
		c.Token = token
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, errors.New("unable to determine user home directory")
		}
		tokenDir := filepath.Join(homeDir, ".time-tracker")
		os.MkdirAll(tokenDir, os.ModePerm)
		tokenPath := filepath.Join(tokenDir, ".token")
		os.WriteFile(tokenPath, []byte(token), os.ModePerm)
	}

	return response, nil
}

// prepareRequest creates a new HTTP request with proper headers for JSON data
func (c *ApiClient) prepareRequest(method, endpoint string, data map[string]interface{}) (*http.Request, error) {
	url := c.BaseURL + endpoint

	var body io.Reader
	contentType := "application/json"

	if data != nil {
		jsonData, jsonErr := json.Marshal(data)
		if jsonErr != nil {
			return nil, fmt.Errorf("failed to marshal request data: %w", jsonErr)
		}
		body = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	req.Header.Set("Content-Type", contentType)

	return req, nil
}

// prepareRequestWithBody creates a new HTTP request with a custom body and content type
func (c *ApiClient) prepareRequestWithBody(method, endpoint string, body io.Reader, contentType string) (*http.Request, error) {
	url := c.BaseURL + endpoint

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	return req, nil
}

func (c *ApiClient) CallAPI(endpoint, method string, data map[string]interface{}) (map[string]interface{}, error) {
	url := c.BaseURL + endpoint

	var req *http.Request
	var err error

	if data != nil {
		jsonData, _ := json.Marshal(data)
		req, err = http.NewRequest(method, url, bytes.NewBuffer(jsonData))
	} else {
		req, err = http.NewRequest(method, url, nil)
	}

	if err != nil {
		return nil, err
	}

	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		println("Unauthorized. Removing token file.")
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, errors.New("unable to determine user home directory")
		}
		tokenPath := filepath.Join(homeDir, ".time-tracker", ".token")
		os.Remove(tokenPath)
		c.Token = ""
		return nil, errors.New("unauthorized")
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, errors.New("API call failed with status: " + resp.Status)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// UploadFile sends a file using multipart/form-data
func (c *ApiClient) UploadFile(endpoint, method, fieldName, fileName string, fileData []byte) (map[string]interface{}, error) {
	url := c.BaseURL + endpoint

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile(fieldName, fileName)
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}
	_, err = io.Copy(part, bytes.NewReader(fileData))
	if err != nil {
		return nil, fmt.Errorf("failed to copy file data: %w", err)
	}

	err = writer.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to close multipart writer: %w", err)
	}

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		println("Unauthorized. Removing token file.")
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, errors.New("unable to determine user home directory")
		}
		tokenPath := filepath.Join(homeDir, ".time-tracker", ".token")
		os.Remove(tokenPath)
		c.Token = ""
		return nil, errors.New("unauthorized")
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API call failed with status: %s, body: %s", resp.Status, string(respBody))
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		// If response is not JSON, maybe return the raw body or handle differently?
		// For now, assume JSON response or return unmarshal error.
		return nil, fmt.Errorf("failed to parse response JSON: %w. Body: %s", err, string(respBody))
	}

	return result, nil
}

// CallAPIForArray makes an API call and expects a JSON array response
func (c *ApiClient) CallAPIForArray(endpoint, method string, data map[string]interface{}) ([]interface{}, error) {
	url := c.BaseURL + endpoint

	var req *http.Request
	var err error

	if data != nil {
		jsonData, _ := json.Marshal(data)
		req, err = http.NewRequest(method, url, bytes.NewBuffer(jsonData))
	} else {
		req, err = http.NewRequest(method, url, nil)
	}

	if err != nil {
		return nil, err
	}

	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		println("Unauthorized. Removing token file.")
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, errors.New("unable to determine user home directory")
		}
		tokenPath := filepath.Join(homeDir, ".time-tracker", ".token")
		os.Remove(tokenPath)
		c.Token = ""
		return nil, errors.New("unauthorized")
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, errors.New("API call failed with status: " + resp.Status)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result []interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	return result, nil
}
