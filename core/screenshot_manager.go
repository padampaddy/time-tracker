package core

import (
	"fmt"
	"image/png"
	"math/rand"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/kbinani/screenshot"
)

type ScreenshotManager struct {
	interval      time.Duration
	isActive      bool
	screenshotDir string
	stopChan      chan struct{}
	wg            sync.WaitGroup
	mu            sync.Mutex
	taskManager   *TaskManager // Added TaskManager reference
}

func NewScreenshotManager(intervalSeconds int, taskManager *TaskManager) *ScreenshotManager {
	// Seed the random number generator (important for randomInterval)
	rand.Seed(time.Now().UnixNano())

	homeDir, _ := os.UserHomeDir()
	screenshotDir := filepath.Join(homeDir, ".time-tracker", "screenshots")
	os.MkdirAll(screenshotDir, os.ModePerm)

	return &ScreenshotManager{
		interval:      time.Duration(intervalSeconds) * time.Second,
		isActive:      false,
		screenshotDir: screenshotDir,
		taskManager:   taskManager,
		// stopChan is initialized in StartCapture
	}
}

func (sm *ScreenshotManager) StartCapture() {
	sm.mu.Lock()
	if sm.isActive {
		sm.mu.Unlock()
		return // Already active, do nothing
	}

	sm.isActive = true
	sm.stopChan = make(chan struct{}) // Initialize channel here
	sm.wg.Add(1)
	go sm.scheduleRandomCapture()
	sm.mu.Unlock()
}

func (sm *ScreenshotManager) StopCapture() {
	sm.mu.Lock()
	// Check if active and channel exists to prevent double close or closing nil channel
	if !sm.isActive || sm.stopChan == nil {
		sm.mu.Unlock()
		return // Not active or already stopped
	}

	// Check if channel is already closed (makes StopCapture idempotent)
	select {
	case <-sm.stopChan:
		// Already closed
	default:
		// Not closed, close it now
		close(sm.stopChan)
	}
	sm.isActive = false // Mark as inactive
	sm.mu.Unlock()      // Unlock BEFORE waiting to prevent deadlock

	sm.wg.Wait() // Wait for the goroutine to finish
}

func (sm *ScreenshotManager) captureScreenshot() (string, error) {
	bounds := screenshot.GetDisplayBounds(0)
	img, err := screenshot.CaptureRect(bounds)
	if err != nil {
		return "", fmt.Errorf("failed to capture screenshot: %w", err)
	}

	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("screenshot_%s.png", timestamp)
	filepath := filepath.Join(sm.screenshotDir, filename)

	file, err := os.Create(filepath)
	if err != nil {
		return "", fmt.Errorf("failed to create screenshot file: %w", err)
	}
	defer file.Close()

	err = png.Encode(file, img)
	if err != nil {
		return "", fmt.Errorf("failed to save screenshot: %w", err)
	}

	// Upload the screenshot if task manager is available
	if sm.taskManager != nil {
		success, err := sm.taskManager.UploadScreenshot(filepath)
		if err != nil {
			fmt.Printf("Failed to upload screenshot: %v\n", err)
		} else if !success {
			fmt.Printf("Screenshot upload was not successful\n")
		}
	}

	return filepath, nil
}

func (sm *ScreenshotManager) scheduleRandomCapture() {
	defer sm.wg.Done() // Ensure Done is called when goroutine exits

	// Use NewTimer for better resource management in loops
	timer := time.NewTimer(sm.randomInterval())
	defer timer.Stop() // Ensure timer resources are cleaned up on exit

	for {
		select {
		case <-sm.stopChan:
			// Stop signal received, exit the loop
			return
		case <-timer.C:
			// Timer fired, capture screenshot
			// No need to check sm.isActive here, stopChan handles termination
			_, err := sm.captureScreenshot()
			if err != nil {
				// Consider using a logger here instead of fmt.Printf
				fmt.Printf("Error capturing screenshot: %s\n", err)
			}
			// Reset the timer for the next random interval
			timer.Reset(sm.randomInterval())
		}
	}
}

func (sm *ScreenshotManager) randomInterval() time.Duration {
	min := float64(sm.interval) * 0.8
	max := float64(sm.interval) * 1.2
	return time.Duration(min + rand.Float64()*(max-min))
}
