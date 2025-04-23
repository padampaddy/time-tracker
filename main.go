package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"github.com/time-tracker/v2/assets"
	"github.com/time-tracker/v2/services"
	"github.com/time-tracker/v2/ui"
)

const tokenFileName = ".token"

// getTokenFilePath returns the path to the token file within a dedicated config directory.
func getTokenFilePath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}
	configDir := filepath.Join(homeDir, ".time-tracker")
	// Ensure the directory exists
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return "", fmt.Errorf("failed to create config directory %s: %w", configDir, err)
	}
	return filepath.Join(configDir, tokenFileName), nil
}

// checkTokenExists checks if the token file exists.
func checkTokenExists() bool {
	tokenPath, err := getTokenFilePath()
	if err != nil {
		log.Printf("Error getting token file path: %v", err)
		return false // Assume no token if path fails
	}
	_, err = os.Stat(tokenPath)
	if os.IsNotExist(err) {
		log.Println("Token file does not exist.")
		return false
	} else if err != nil {
		log.Printf("Error checking token file %s: %v", tokenPath, err)
		return false // Assume no token on error
	}
	log.Println("Token file found.")
	return true
}

// saveToken saves the token to the designated file.
func saveToken(token string) error {
	tokenPath, err := getTokenFilePath()
	if err != nil {
		return fmt.Errorf("failed to get token file path for saving: %w", err)
	}
	// Write the token, overwriting the file if it exists.
	// Set permissions to be readable/writable only by the user.
	err = os.WriteFile(tokenPath, []byte(token), 0600)
	if err != nil {
		return fmt.Errorf("failed to write token file %s: %w", tokenPath, err)
	}
	log.Printf("Token saved successfully to %s", tokenPath)
	return nil
}

// showTaskWindow creates and displays the main task window.
func showTaskWindow(a fyne.App) {
	log.Println("Showing Task Window...")
	// We pass the app instance to the task window constructor
	taskUI := ui.NewTaskWindow(a)
	// The Run method of TaskWindowUI likely calls a.Run() or manages its own window showing.
	// If NewTaskWindow just creates the window, we need to show it.
	// Let's assume NewTaskWindow prepares it and we just need to show the window.
	taskUI.Win.Show()
}

func main() {
	// Initialize the Fyne application
	myApp := app.New()

	// Set the application icon using the embedded resource
	iconResource := assets.GetClockResource()
	if iconResource == nil {
		log.Println("Failed to load icon from embedded resources")
	} else {
		myApp.SetIcon(iconResource)
	}

	// Initialize the authentication service
	// Assuming NewAuthService() exists and is correctly implemented
	authSvc := services.NewAuthService() // You might need to pass config here

	// Check if the token exists
	if checkTokenExists() {
		// Token exists, show the main task window directly
		log.Println("Token exists, launching main application.")
		showTaskWindow(myApp)
	} else {
		// Token does not exist, show the login window
		log.Println("Token does not exist, launching login window.")

		// Define the callback function for successful login
		onLoginSuccess := func(token string) {
			log.Println("Login successful, proceeding to main application.")
			// Save the received token
			err := saveToken(token)
			if err != nil {
				log.Printf("FATAL: Failed to save token: %v", err)
				// Decide how to handle this - maybe show an error dialog?
				// For now, we log fatally, but a real app might retry or show UI error.
				// We might not want to proceed without saving the token.
				// However, for now, let's proceed to show the task window anyway.
				// Consider adding dialog.ShowError(err, currentWindow) if possible.
			}
			// Show the main task window
			showTaskWindow(myApp)
		}

		// Create and show the login window, passing the app, service, and success callback
		// The login window will close itself upon successful login via the callback.
		loginWin := ui.NewLoginWindow(myApp, authSvc, onLoginSuccess)
		loginWin.Show()
	}

	// Start the Fyne application event loop.
	// This will block until the application exits (e.g., user quits from tray or closes last window if not configured otherwise).
	myApp.Run()
	log.Println("Application has exited.")
}
