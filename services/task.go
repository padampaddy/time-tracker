package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"log"
	"mime/multipart"
	"net/http"

	"github.com/time-tracker/v2/internal/config"
	"github.com/time-tracker/v2/internal/types"
)

// TaskService handles task-related operations
type TaskService struct {
	apiClient *ApiClient
}

// NewTaskService creates a new instance of TaskService
func NewTaskService() *TaskService {
	return &TaskService{
		apiClient: NewApiClient(config.API_URL),
	}
}

// GetUserTasks fetches all tasks for the authenticated user
func (s *TaskService) GetUserTasks() ([]types.Task, error) {
	response, err := s.apiClient.CallAPIForArray("/api/tasks/user", "GET", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch tasks: %w", err)
	}

	jsonData, err := json.Marshal(response)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response: %w", err)
	}

	var tasks []types.Task
	if err := json.Unmarshal(jsonData, &tasks); err != nil {
		return nil, fmt.Errorf("failed to parse task data: %w", err)
	}

	return tasks, nil
}

// StartUserTask starts a user task by creating a work report
func (s *TaskService) StartUserTask(projectID, taskID int, description string, startTime string) (*types.WorkReport, error) {
	payload := map[string]interface{}{
		"project":     projectID,
		"task":        taskID,
		"description": description,
		"start_time":  startTime,
	}

	response, err := s.apiClient.CallAPI("/api/work_report", "POST", payload)
	if err != nil {
		return nil, fmt.Errorf("failed to start task: %w", err)
	}

	jsonData, err := json.Marshal(response)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response: %w", err)
	}

	var workReport types.WorkReport
	if err := json.Unmarshal(jsonData, &workReport); err != nil {
		return nil, fmt.Errorf("failed to parse work report: %w", err)
	}

	return &workReport, nil
}

// StopUserTask stops a user task by updating the work report with an end time
func (s *TaskService) StopUserTask(workReportID int, endTime string, description *string) (*types.WorkReport, error) {
	payload := map[string]interface{}{
		"end_time": endTime,
	}
	if description != nil {
		payload["description"] = *description
	}

	response, err := s.apiClient.CallAPI(fmt.Sprintf("/api/work_report/%d", workReportID), "PUT", payload)
	if err != nil {
		return nil, fmt.Errorf("failed to stop task: %w", err)
	}

	jsonData, err := json.Marshal(response)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response: %w", err)
	}

	var workReport types.WorkReport
	if err := json.Unmarshal(jsonData, &workReport); err != nil {
		return nil, fmt.Errorf("failed to parse work report: %w", err)
	}

	return &workReport, nil
}

// UploadScreenshot uploads a screenshot and webcam image for a specific work report
func (s *TaskService) UploadScreenshot(workReportID int, screenshotData []byte, filename string) error {
	// Construct the API endpoint URL
	url := fmt.Sprintf("/api/upload_image/%d", workReportID)

	// Prepare the multipart form data
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add the screenshot file part
	part, err := writer.CreateFormFile("screenshot", filename)
	if err != nil {
		return fmt.Errorf("failed to create form file: %w", err)
	}
	_, err = io.Copy(part, bytes.NewReader(screenshotData))
	if err != nil {
		return fmt.Errorf("failed to copy screenshot data: %w", err)
	}

	// Add the webcam image file part
	webcamPart, err := writer.CreateFormFile("webcam_image", "webcam.png")
	if err != nil {
		return fmt.Errorf("failed to create webcam form file: %w", err)
	}
	_, err = io.Copy(webcamPart, bytes.NewReader(createBlackPNG()))
	if err != nil {
		return fmt.Errorf("failed to copy webcam image data: %w", err)
	}

	// Close the multipart writer
	err = writer.Close()
	if err != nil {
		return fmt.Errorf("failed to close multipart writer: %w", err)
	}

	// Prepare the request using the new function
	contentType := writer.FormDataContentType()
	req, err := s.apiClient.prepareRequestWithBody("POST", url, body, contentType)
	if err != nil {
		return fmt.Errorf("failed to prepare request: %w", err)
	}

	// Execute the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to upload screenshot: %w", err)
	}
	defer resp.Body.Close()

	// Check the response status code
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body) // Read body for error details
		return fmt.Errorf("screenshot upload failed with status %s: %s", resp.Status, string(respBody))
	}

	// Screenshot uploaded successfully
	return nil
}

// createBlackPNG generates a 100x100 all-black PNG image and returns its byte representation
func createBlackPNG() []byte {
	const width, height = 100, 100 // Dimensions of the black PNG

	// Create a black image
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.Black)
		}
	}

	// Encode the image to PNG format
	buf := &bytes.Buffer{}
	err := png.Encode(buf, img)
	if err != nil {
		log.Fatalf("failed to encode black PNG: %v", err)
	}

	return buf.Bytes()
}
