package core

import (
	"time"
)

type Activity struct {
	TaskName  string    `json:"task_name"`
	Timestamp time.Time `json:"timestamp"`
}

type ActivityTracker struct {
	ActiveTasks       []Activity
	IsTracking        bool
	CurrentTask       *string
	StartTime         *time.Time
	EndTime           *time.Time
	Database          *Database
	ScreenshotManager *ScreenshotManager
	InputMonitor      *InputMonitor
	screenshotDir     string
	taskManager       *TaskManager // Added TaskManager field
}

// Updated NewActivityTracker to accept TaskManager
func NewActivityTracker(screenshotDir string, taskManager *TaskManager) *ActivityTracker {
	return &ActivityTracker{
		ActiveTasks:       []Activity{},
		IsTracking:        false,
		CurrentTask:       nil,
		StartTime:         nil,
		EndTime:           nil,
		Database:          NewDatabase("time_tracker.db"),
		ScreenshotManager: NewScreenshotManager(10, taskManager),
		InputMonitor:      NewInputMonitor(),
		screenshotDir:     screenshotDir,
		taskManager:       taskManager,
	}
}

func (at *ActivityTracker) StartTracking(taskName string) error {
	err := at.Database.Connect()
	if err != nil {
		return err
	}
	at.IsTracking = true
	at.CurrentTask = &taskName
	now := time.Now()
	at.StartTime = &now
	at.ScreenshotManager.StartCapture()
	at.InputMonitor.StartMonitoring()
	return at.trackActivities()
}

func (at *ActivityTracker) StopTracking() error {
	at.IsTracking = false
	at.CurrentTask = nil
	now := time.Now()
	at.EndTime = &now
	err := at.trackActivities()
	if err != nil {
		return err
	}
	err = at.saveCurrentSession()
	if err != nil {
		return err
	}
	at.ScreenshotManager.StopCapture()
	at.InputMonitor.StopMonitoring() // Stop input monitoring when tracking stops
	return nil
}

func (at *ActivityTracker) GetActiveTasks() []Activity {
	return at.ActiveTasks
}

func (at *ActivityTracker) LogActivity(taskName string) {
	activity := Activity{
		TaskName:  taskName,
		Timestamp: time.Now(),
	}
	at.ActiveTasks = append(at.ActiveTasks, activity)
}

func (at *ActivityTracker) GetLoggedActivities() []Activity {
	return at.ActiveTasks
}

func (at *ActivityTracker) trackActivities() error {
	if at.CurrentTask != nil {
		at.LogActivity(*at.CurrentTask)
	}
	return nil
}

func (at *ActivityTracker) saveCurrentSession() error {
	duration := at.calculateSessionDuration()
	// Use screenshotDir to save the screenshot
	screenshotPath, err := at.ScreenshotManager.captureScreenshot()
	if err != nil {
		// Allow continuing even if screenshot fails, just log it or handle differently
		screenshotPath = "" // Or some indicator that screenshot failed
	}
	// Get counts without stopping again
	for _, activity := range at.ActiveTasks {
		// Ensure StartTime and EndTime are not nil before formatting
		startTimeStr := ""
		if at.StartTime != nil {
			startTimeStr = at.StartTime.Format(time.RFC3339)
		}
		endTimeStr := ""
		if at.EndTime != nil {
			endTimeStr = at.EndTime.Format(time.RFC3339)
		}

		err := at.Database.SaveActivity(
			activity.TaskName,
			startTimeStr,
			endTimeStr,
			int(duration),
			screenshotPath,
			0, 0)
		if err != nil {
			return err // Or collect errors and return aggregate
		}
	}
	at.ActiveTasks = []Activity{} // Clear active tasks after saving
	return nil
}

func (at *ActivityTracker) calculateSessionDuration() float64 {
	if at.StartTime != nil && at.EndTime != nil {
		return at.EndTime.Sub(*at.StartTime).Seconds()
	}
	return 0.0
}
