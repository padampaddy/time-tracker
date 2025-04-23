package assets

import (
	"embed"

	"fyne.io/fyne/v2"
)

//go:embed clock.png clock.ico
var assetsFS embed.FS

// GetClockPNG returns the clock.png resource for Fyne
func GetClockResource() fyne.Resource {
	data, err := assetsFS.ReadFile("clock.png")
	if err != nil {
		return nil
	}
	return fyne.NewStaticResource("clock.png", data)
}

// GetClockIconResource returns the clock.ico resource for Fyne
func GetClockIconResource() fyne.Resource {
	data, err := assetsFS.ReadFile("clock.ico")
	if err != nil {
		return nil
	}
	return fyne.NewStaticResource("clock.ico", data)
}
