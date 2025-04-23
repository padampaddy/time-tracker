package ui

import (
	"fmt"
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/time-tracker/v2/internal/auth"
)

var authService auth.Service

// SetAuthService allows dependency injection of the auth service
func SetAuthService(service auth.Service) {
	authService = service
}

// NewLoginWindow creates and returns the login window.
// It calls the onSuccess callback with the user's token upon successful login.
func NewLoginWindow(a fyne.App, service auth.Service, onSuccess func(token string)) fyne.Window {
	if service == nil {
		log.Fatal("Auth service not provided to NewLoginWindow")
	}
	authService = service // Keep the package-level variable for consistency if needed elsewhere, or remove if only used here.

	win := a.NewWindow("Login")

	emailEntry := widget.NewEntry()
	emailEntry.SetPlaceHolder("Email")

	passwordEntry := widget.NewPasswordEntry()
	passwordEntry.SetPlaceHolder("Password")

	statusLabel := widget.NewLabel("") // To show error messages

	loginButton := widget.NewButton("Login", func() {
		statusLabel.SetText("Logging in...") // Provide feedback
		email := emailEntry.Text
		password := passwordEntry.Text

		if email == "" || password == "" {
			log.Println("Email and password cannot be empty")
			statusLabel.SetText("Email and password required.")
			dialog.ShowError(fmt.Errorf("email and password cannot be empty"), win) // Show dialog too
			return
		}

		// Assume Login returns a user object with a Token field and an error
		// Adjust this based on the actual signature of authService.Login
		user, err := authService.Login(email, password)
		if err != nil {
			log.Printf("Login failed: %v", err)
			statusLabel.SetText("Login failed: " + err.Error())
			dialog.ShowError(err, win) // Show specific error
			return
		}

		// Assuming successful login returns a non-nil user with a token
		// Adjust the condition and token access based on your authService implementation
		if user != nil && user.Token != "" { // Example: Check for user and token
			log.Printf("Login successful for user: %s", user.Username) // Assuming user has Username
			statusLabel.SetText("Login successful!")
			// Call the success callback with the token
			onSuccess(user.Token) // Pass the token
			win.Close()           // Close the login window
		} else {
			// Handle cases where login might succeed but return no user/token, or specific errors
			log.Println("Login failed: Invalid credentials or unexpected response.")
			statusLabel.SetText("Invalid email or password.")
			dialog.ShowError(fmt.Errorf("invalid email or password"), win)
		}
	})

	form := container.NewVBox(
		widget.NewLabel("Please Log In"),
		emailEntry,
		passwordEntry,
		loginButton,
		statusLabel, // Add status label to the form
	)

	win.SetContent(form)
	win.Resize(fyne.NewSize(300, 200))
	win.SetFixedSize(true) // Prevent resizing
	win.CenterOnScreen()   // Center the login window
	return win
}
