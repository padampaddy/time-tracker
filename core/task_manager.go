package core

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/time-tracker/v2/internal/types"
	"github.com/time-tracker/v2/services"
)

type TaskManager struct {
	tasks       []types.Task
	activeTask  *types.Task
	taskHistory map[int][]map[string]interface{}
	taskService *services.TaskService
	workReport  *types.WorkReport
}

func NewTaskManager() *TaskManager {
	return &TaskManager{
		tasks:       []types.Task{},
		activeTask:  nil,
		taskHistory: make(map[int][]map[string]interface{}),
		taskService: services.NewTaskService(),
	}
}

func (tm *TaskManager) AddTask(task types.Task) (bool, error) {
	for _, t := range tm.tasks {
		if t.ID == task.ID {
			return false, nil
		}
	}
	tm.tasks = append(tm.tasks, task)
	tm.taskHistory[task.ID] = []map[string]interface{}{}
	return true, nil
}

func (tm *TaskManager) RemoveTask(task types.Task) (bool, error) {
	for i, t := range tm.tasks {
		if t.ID == task.ID {
			tm.tasks = append(tm.tasks[:i], tm.tasks[i+1:]...)
			if tm.activeTask != nil && tm.activeTask.ID == task.ID {
				tm.activeTask = nil
			}
			return true, nil
		}
	}
	return false, nil
}

func (tm *TaskManager) GetTasks() ([]types.Task, error) {
	tasks, err := tm.taskService.GetUserTasks()
	if err != nil {
		return nil, err
	}
	tm.tasks = tasks
	return tm.tasks, nil
}

func (tm *TaskManager) ClearTasks() {
	tm.tasks = []types.Task{}
	tm.activeTask = nil
	tm.taskHistory = make(map[int][]map[string]interface{})
}

func (tm *TaskManager) SetActiveTask(task types.Task) (bool, error) {
	for _, t := range tm.tasks {
		if t.ID == task.ID {
			tm.activeTask = &task
			tm.taskHistory[task.ID] = append(tm.taskHistory[task.ID], map[string]interface{}{
				"start_time": time.Now(),
				"end_time":   nil,
			})
			return true, nil
		}
	}
	return false, nil
}

func (tm *TaskManager) StopActiveTask() {
	if tm.activeTask != nil {
		history := tm.taskHistory[tm.activeTask.ID]
		if len(history) > 0 {
			lastSession := history[len(history)-1]
			if lastSession["end_time"] == nil {
				lastSession["end_time"] = time.Now()
			}
		}
		tm.activeTask = nil
	}
}

func (tm *TaskManager) GetActiveTask() *types.Task {
	return tm.activeTask
}

func (tm *TaskManager) GetTaskHistory(task types.Task) []map[string]interface{} {
	return tm.taskHistory[task.ID]
}

func (tm *TaskManager) UserStartTask(projectID int, task types.Task, description string) (bool, error) {
	if tm.activeTask != nil {
		tm.StopActiveTask()
	}

	startTime := time.Now().Format(time.RFC3339)
	workReport, err := tm.taskService.StartUserTask(projectID, task.ID, description, startTime)
	if err != nil {
		return false, err
	}

	tm.workReport = workReport
	if tm.workReport != nil {
		tm.activeTask = &task
		tm.taskHistory[task.ID] = append(tm.taskHistory[task.ID], map[string]interface{}{
			"start_time":  startTime,
			"end_time":    nil,
			"description": description,
		})
		return true, nil
	}
	return false, nil
}

func (tm *TaskManager) UserStopTask(description string) (bool, error) {
	if tm.workReport == nil || tm.activeTask == nil {
		return false, errors.New("no active task to stop")
	}

	endTime := time.Now().Format(time.RFC3339)
	updatedReport, err := tm.taskService.StopUserTask(tm.workReport.ID, endTime, &description)
	if err != nil {
		return false, err
	}

	if updatedReport != nil {
		history := tm.taskHistory[tm.activeTask.ID]
		lastSession := history[len(history)-1]
		lastSession["end_time"] = endTime
		lastSession["description"] = &description
		tm.activeTask = nil
		return true, nil
	}
	return false, nil
}

// UploadScreenshot uploads a screenshot for a specific work report.
func (tm *TaskManager) UploadScreenshot(filePath string) (bool, error) {
	if tm.workReport == nil {
		return false, nil // Silently skip upload if no active work report
	}

	// Read the file data
	fileData, err := os.ReadFile(filePath)
	if err != nil {
		return false, fmt.Errorf("failed to read screenshot file: %w", err)
	}

	// Get the filename from the path
	filename := filepath.Base(filePath)

	// Call the taskService to upload the screenshot
	err = tm.taskService.UploadScreenshot(tm.workReport.ID, fileData, filename)
	if err != nil {
		return false, err
	}
	return true, nil
}
