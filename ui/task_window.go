package ui

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/time-tracker/v2/core"
	"github.com/time-tracker/v2/internal/types"
)

// TaskWindowUI holds the Fyne UI elements corresponding to the Python TaskWindow

type TaskWindowUI struct {
	App fyne.App
	Win fyne.Window

	taskSelect       *widget.Select
	refreshButton    *widget.Button
	timerLabel       *widget.Label
	startButton      *widget.Button
	stopButton       *widget.Button
	statusLabel      *widget.Label
	screenshotsBox   *fyne.Container
	openFolderButton *widget.Button

	ticker         *time.Ticker
	stopTicker     chan bool
	elapsedTime    time.Duration
	isTimerRunning bool

	tasks           []types.Task
	selectedTask    *types.Task
	screenshotDir   string
	taskManager     *core.TaskManager
	activityTracker *core.ActivityTracker
}

// NewTaskWindow creates and initializes the Fyne UI
func NewTaskWindow(a fyne.App) *TaskWindowUI {
	ui := &TaskWindowUI{
		App:        a,
		stopTicker: make(chan bool),
	}
	ui.Win = a.NewWindow("Go Time Tracker")
	ui.Win.Resize(fyne.NewSize(400, 560))
	ui.Win.SetFixedSize(true)

	iconResource, err := fyne.LoadResourceFromPath("assets/clock.png")
	if err != nil {
		log.Printf("Error loading icon: %v", err)
	} else {
		ui.Win.SetIcon(iconResource)
	}
	ui.taskManager = core.NewTaskManager()
	homeDir, _ := os.UserHomeDir()
	ui.screenshotDir = filepath.Join(homeDir, ".time-tracker", "screenshots")
	os.MkdirAll(ui.screenshotDir, os.ModePerm)

	ui.activityTracker = core.NewActivityTracker(ui.screenshotDir, ui.taskManager)
	ui.setupUI()
	ui.loadTasks()

	ui.Win.SetCloseIntercept(func() {
		ui.Win.Hide()
	})

	ui.setupSystemTray()

	return ui
}

// setupUI creates the main layout and widgets
func (ui *TaskWindowUI) setupUI() {
	ui.taskSelect = widget.NewSelect([]string{"Loading tasks..."}, func(s string) {
		for i := range ui.tasks {
			taskDisplay := fmt.Sprintf("%s (ID: %d, Project: %s)", ui.tasks[i].Name, ui.tasks[i].ID, ui.tasks[i].Project.Name)
			if taskDisplay == s {
				ui.selectedTask = &ui.tasks[i]
				log.Printf("Selected task: %s (ID: %d)", ui.selectedTask.Name, ui.selectedTask.ID)
				break
			}
		}
	})
	ui.refreshButton = widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), ui.loadTasks)
	taskSelectionLayout := container.NewBorder(nil, nil, nil, ui.refreshButton, ui.taskSelect)
	taskCard := widget.NewCard("Task Selection", "", taskSelectionLayout)

	ui.timerLabel = widget.NewLabel("00:00:00")
	ui.timerLabel.Alignment = fyne.TextAlignCenter
	ui.timerLabel.TextStyle = fyne.TextStyle{Bold: true, Monospace: true}
	ui.timerLabel.Importance = widget.HighImportance

	ui.startButton = widget.NewButton("Start Timer", ui.startTimer)
	ui.stopButton = widget.NewButton("Stop Timer", ui.stopTimer)
	ui.stopButton.Disable()
	timerButtons := container.NewGridWithColumns(2, ui.startButton, ui.stopButton)
	timerLayout := container.NewVBox(ui.timerLabel, timerButtons)
	timerCard := widget.NewCard("Timer Controls", "", timerLayout)

	ui.statusLabel = widget.NewLabel("No task active")
	ui.statusLabel.Alignment = fyne.TextAlignCenter
	statusCard := widget.NewCard("Current Status", "", container.NewCenter(ui.statusLabel))

	ui.screenshotsBox = container.NewHBox()
	scrollContainer := container.NewHScroll(ui.screenshotsBox)
	scrollContainer.SetMinSize(fyne.NewSize(380, 120))

	ui.openFolderButton = widget.NewButton("Open Screenshots Folder", ui.openScreenshotsFolder)
	screenshotLayout := container.NewVBox(scrollContainer, ui.openFolderButton)
	screenshotCard := widget.NewCard("Recent Screenshots", "", screenshotLayout)
	ui.updateScreenshotsList()

	content := container.NewVBox(
		taskCard,
		timerCard,
		statusCard,
		screenshotCard,
		layout.NewSpacer(),
	)
	ui.Win.SetContent(content)
}

// loadTasks fetches tasks (placeholder) and updates the dropdown
func (ui *TaskWindowUI) loadTasks() {
	ui.taskSelect.Disable()
	ui.refreshButton.Disable()
	ui.taskSelect.PlaceHolder = "Refreshing..."
	ui.taskSelect.Refresh()

	go func() {
		time.Sleep(500 * time.Millisecond)
		tasks, err := ui.taskManager.GetTasks()
		fyne.Do(func() {
			if err != nil {
				log.Printf("Error loading tasks: %v", err)
				ui.taskSelect.PlaceHolder = "Error loading tasks"
				ui.taskSelect.Refresh()
				return
			}
			ui.tasks = tasks
			taskDisplays := make([]string, len(ui.tasks))
			for i, task := range ui.tasks {
				taskDisplays[i] = fmt.Sprintf("%s (ID: %d, Project: %s)", task.Name, task.ID, task.Project.Name)
			}

			if len(taskDisplays) == 0 {
				taskDisplays = []string{"No tasks found"}
				ui.taskSelect.PlaceHolder = "No tasks found"
			} else {
				ui.taskSelect.PlaceHolder = "Select a task..."
			}

			ui.taskSelect.Options = taskDisplays
			fyne.Do(func() {
				ui.taskSelect.ClearSelected()
				ui.selectedTask = nil
				ui.taskSelect.Enable()
				ui.refreshButton.Enable()
				ui.taskSelect.Refresh()
				log.Println("Tasks refreshed")
			})
		})
	}()
}

// startTimer handles the start button click
func (ui *TaskWindowUI) startTimer() {
	if ui.selectedTask == nil {
		dialog.ShowError(fmt.Errorf("please select a task first"), ui.Win)
		return
	}
	if ui.isTimerRunning {
		return
	}

	log.Printf("Starting timer and activity tracking for task: %s", ui.selectedTask.Name)

	err := ui.activityTracker.StartTracking(ui.selectedTask.Name)
	if err != nil {
		log.Printf("Error starting activity tracker: %v", err)
		dialog.ShowError(fmt.Errorf("failed to start tracking: %w", err), ui.Win)
		return
	}

	ui.isTimerRunning = true
	ui.elapsedTime = 0
	ui.ticker = time.NewTicker(1 * time.Second)
	ui.stopTicker = make(chan bool)
	ui.taskManager.SetActiveTask(*ui.selectedTask)
	go ui.taskManager.UserStartTask(ui.selectedTask.Project.ID, *ui.selectedTask, "Started")
	go func() {
		for {
			select {
			case <-ui.ticker.C:
				ui.elapsedTime += time.Second
				ui.updateTimerDisplay()
			case <-ui.stopTicker:
				ui.ticker.Stop()
				log.Println("Timer stopped goroutine exiting.")
				return
			}
		}
	}()

	ui.updateUIForStart()
}

// stopTimer handles the stop button click
func (ui *TaskWindowUI) stopTimer() {
	if !ui.isTimerRunning {
		return
	}

	log.Println("Stopping timer and activity tracking")

	err := ui.activityTracker.StopTracking()
	if err != nil {
		log.Printf("Error stopping activity tracker: %v", err)
		dialog.ShowError(fmt.Errorf("failed to properly stop tracking session: %w", err), ui.Win)
	}
	go ui.taskManager.UserStopTask("Stopped")

	go func() {
		if ui.ticker != nil {
			close(ui.stopTicker)
		}
		fyne.Do(func() {
			ui.isTimerRunning = false
			ui.updateUIForStop()
			ui.timerLabel.SetText("00:00:00")
			ui.updateScreenshotsList()
		})
	}()
}

// updateTimerDisplay updates the timer label text
func (ui *TaskWindowUI) updateTimerDisplay() {
	hours := int(ui.elapsedTime.Hours())
	minutes := int(ui.elapsedTime.Minutes()) % 60
	seconds := int(ui.elapsedTime.Seconds()) % 60
	fyne.Do(func() {
		ui.timerLabel.SetText(fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds))
	})
}

// updateUIForStart adjusts widget states when timer starts
func (ui *TaskWindowUI) updateUIForStart() {
	ui.startButton.Disable()
	ui.stopButton.Enable()
	ui.taskSelect.Disable()
	ui.refreshButton.Disable()
	if ui.selectedTask != nil {
		ui.statusLabel.SetText(fmt.Sprintf("Tracking: %s", ui.selectedTask.Name))
	} else {
		ui.statusLabel.SetText("Tracking: Unknown Task")
	}
}

// updateUIForStop adjusts widget states when timer stops
func (ui *TaskWindowUI) updateUIForStop() {
	ui.startButton.Enable()
	ui.stopButton.Disable()
	ui.taskSelect.Enable()
	ui.refreshButton.Enable()
	ui.statusLabel.SetText("No task active")
}

// updateScreenshotsList loads recent screenshots and displays them
func (ui *TaskWindowUI) updateScreenshotsList() {
	ui.screenshotsBox.RemoveAll()

	go func() {
		files, err := os.ReadDir(ui.screenshotDir)
		fyne.Do(func() {
			if err != nil {
				log.Printf("Error reading screenshot dir: %v", err)
				ui.screenshotsBox.Add(widget.NewLabel("Error loading screenshots"))
				ui.screenshotsBox.Refresh()
				return
			}

			type fileInfo struct {
				path    string
				modTime time.Time
			}
			var screenshots []fileInfo

			for _, file := range files {
				if !file.IsDir() && strings.HasPrefix(file.Name(), "screenshot_") && strings.HasSuffix(file.Name(), ".png") {
					info, err := file.Info()
					if err == nil {
						screenshots = append(screenshots, fileInfo{
							path:    filepath.Join(ui.screenshotDir, file.Name()),
							modTime: info.ModTime(),
						})
					}
				}
			}

			sort.Slice(screenshots, func(i, j int) bool {
				return screenshots[i].modTime.After(screenshots[j].modTime)
			})

			limit := 5
			if len(screenshots) < limit {
				limit = len(screenshots)
			}

			if limit == 0 {
				ui.screenshotsBox.Add(widget.NewLabel("No screenshots yet."))
			} else {
				for i := 0; i < limit; i++ {
					ssPath := screenshots[i].path

					timestampStr := "Unknown time"
					nameOnly := strings.TrimSuffix(filepath.Base(ssPath), ".png")
					parts := strings.Split(nameOnly, "_")
					if len(parts) == 3 {
						ts, err := time.Parse("20060102_150405", parts[1]+"_"+parts[2])
						if err == nil {
							timestampStr = ts.Format("Jan 02, 2006 03:04 PM")
						}
					}

					img := canvas.NewImageFromFile(ssPath)
					if img == nil {
						log.Printf("Warning: Failed to load image %s", ssPath)
						img = canvas.NewImageFromResource(theme.BrokenImageIcon())
					}
					img.FillMode = canvas.ImageFillContain
					img.SetMinSize(fyne.NewSize(100, 100))

					imgButton := widget.NewButton("", func() { ui.openScreenshotPreview(ssPath) })
					imgButton.Importance = widget.LowImportance
					clickableImage := container.NewStack(imgButton, img)

					timestampLabel := widget.NewLabel(timestampStr)
					timestampLabel.Wrapping = fyne.TextWrapOff
					timestampLabel.Alignment = fyne.TextAlignCenter
					timestampLabel.Importance = widget.LowImportance

					screenshotItem := container.New(layout.NewVBoxLayout(),
						clickableImage,
						timestampLabel,
					)
					ui.screenshotsBox.Add(screenshotItem)
				}
			}

			ui.screenshotsBox.Refresh()
		})
	}()
}

// openScreenshotPreview opens a specific screenshot file
func (ui *TaskWindowUI) openScreenshotPreview(path string) {
	go func() {
		uri := storage.NewFileURI(path)
		parsedURL, err := url.Parse(uri.String())
		fyne.Do(func() {
			if err != nil {
				log.Printf("Failed to parse screenshot URI %s: %v", uri.String(), err)
				dialog.ShowError(fmt.Errorf("invalid screenshot path"), ui.Win)
				return
			}
			err = ui.App.OpenURL(parsedURL)
			if err != nil {
				log.Printf("Failed to open screenshot %s: %v", path, err)
				dialog.ShowError(fmt.Errorf("could not open screenshot viewer: %w", err), ui.Win)
			}
		})
	}()
}

// openScreenshotsFolder opens the directory containing screenshots
func (ui *TaskWindowUI) openScreenshotsFolder() {
	go func() {
		uri := storage.NewFileURI(ui.screenshotDir)
		parsedURL, err := url.Parse(uri.String())
		fyne.Do(func() {
			if err != nil {
				log.Printf("Failed to parse screenshot folder URI %s: %v", uri.String(), err)
				dialog.ShowError(fmt.Errorf("invalid screenshot folder path"), ui.Win)
				return
			}
			err = ui.App.OpenURL(parsedURL)
			if err != nil {
				log.Printf("Failed to open screenshot folder %s: %v", ui.screenshotDir, err)
				dialog.ShowError(fmt.Errorf("could not open file explorer: %w", err), ui.Win)
			}
		})
	}()
}

// setupSystemTray configures the system tray icon and menu
func (ui *TaskWindowUI) setupSystemTray() {
	if desk, ok := ui.App.(desktop.App); ok {
		showMenuItem := fyne.NewMenuItem("Show", func() {
			ui.Win.Show()
			ui.Win.RequestFocus()
		})

		menu := fyne.NewMenu("Time Tracker", showMenuItem)
		desk.SetSystemTrayMenu(menu)

		iconResource, err := fyne.LoadResourceFromPath("assets/clock.png")
		if err != nil {
			log.Printf("Error loading system tray icon: %v", err)
		} else {
			desk.SetSystemTrayIcon(iconResource)
		}
	} else {
		log.Println("System tray not supported on this platform.")
	}
}

// Run starts the Fyne application event loop
func (ui *TaskWindowUI) Run() {
	ui.Win.Show()
	ui.App.Run()
	log.Println("Application finished.")
}
