package core

import (
	"fmt"
	"sync"
	"time"

	hook "github.com/robotn/gohook"
)

type InputEvent struct {
	EventType string    // "press", "click", "scroll"
	Key       string    // Key pressed (for keyboard events)
	Position  [2]int    // Mouse position (for mouse events)
	Button    string    // Mouse button (for click events)
	Pressed   bool      // Whether the button is pressed (for click events)
	Scroll    [2]int    // Scroll delta (for scroll events)
	Timestamp time.Time // Event timestamp
}

type InputMonitor struct {
	Keystrokes     []InputEvent
	MouseMovements []InputEvent
	IsMonitoring   bool
	mu             sync.Mutex
}

func NewInputMonitor() *InputMonitor {
	return &InputMonitor{
		Keystrokes:     []InputEvent{},
		MouseMovements: []InputEvent{},
		IsMonitoring:   false,
	}
}

func (im *InputMonitor) StartMonitoring() {
	im.mu.Lock()

	if im.IsMonitoring {
		im.mu.Unlock()
		return
	}

	im.IsMonitoring = true
	im.mu.Unlock() // Unlock before starting the long-running hook

	// Start event monitoring in a separate goroutine
	go func() {
		evChan := hook.Start()
		defer hook.End()

		for {
			im.mu.Lock()
			isMonitoring := im.IsMonitoring
			im.mu.Unlock()

			if !isMonitoring {
				break // Exit goroutine if monitoring stopped
			}

			select {
			case ev := <-evChan:
				im.mu.Lock()
				if !im.IsMonitoring { // Double check after receiving event
					im.mu.Unlock()
					break
				}
				switch ev.Kind {
				case hook.KeyDown, hook.KeyHold:
					keyStr := fmt.Sprintf("%c", ev.Keychar) // Convert rune to string
					// You might want more sophisticated key mapping here
					// For special keys, ev.Rawcode and ev.Keycode might be useful
					inputEvent := InputEvent{
						EventType: "press",
						Key:       keyStr,
						Timestamp: time.Now(),
					}
					im.Keystrokes = append(im.Keystrokes, inputEvent)
				case hook.MouseDown:
					var button string
					switch ev.Button {
					case hook.MouseMap["left"]:
						button = "left"
					case hook.MouseMap["right"]:
						button = "right"
					case hook.MouseMap["middle"]:
						button = "middle"
					default:
						button = "other"
					}
					inputEvent := InputEvent{
						EventType: "click",
						Button:    button,
						Pressed:   true, // gohook only provides MouseDown, not Up
						Timestamp: time.Now(),
					}
					im.MouseMovements = append(im.MouseMovements, inputEvent)
				case hook.MouseWheel:
					// ev.Rotation > 0 is wheel down, < 0 is wheel up
					// ev.Amount seems to indicate lines scrolled
					var scrollY int
					if ev.Rotation > 0 {
						scrollY = -int(ev.Amount) // Down
					} else {
						scrollY = int(ev.Amount) // Up
					}
					inputEvent := InputEvent{
						EventType: "scroll",
						Scroll:    [2]int{0, scrollY},
						Timestamp: time.Now(),
					}
					im.MouseMovements = append(im.MouseMovements, inputEvent)
				}
				im.mu.Unlock()
			case <-time.After(100 * time.Millisecond): // Check periodically if monitoring stopped
				continue
			}
		}
	}()
}

func (im *InputMonitor) StopMonitoring() map[string]int {
	im.mu.Lock()
	defer im.mu.Unlock()

	if !im.IsMonitoring {
		return map[string]int{
			"keyboard_event_count": 0,
			"mouse_event_count":    0,
		}
	}

	im.IsMonitoring = false
	// hook.End() is called in the goroutine when IsMonitoring becomes false

	eventCounts := map[string]int{
		"keyboard_event_count": len(im.Keystrokes),
		"mouse_event_count":    len(im.MouseMovements),
	}

	// Clear data after stopping
	im.ClearData()

	return eventCounts
}

func (im *InputMonitor) ClearData() {
	im.Keystrokes = []InputEvent{}
	im.MouseMovements = []InputEvent{}
}

func (im *InputMonitor) GetKeystrokes() []InputEvent {
	im.mu.Lock()
	defer im.mu.Unlock()
	return im.Keystrokes
}

func (im *InputMonitor) GetMouseMovements() []InputEvent {
	im.mu.Lock()
	defer im.mu.Unlock()
	return im.MouseMovements
}
