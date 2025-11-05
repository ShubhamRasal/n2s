package components

import (
	"github.com/rivo/tview"
)

// Footer represents the application footer component showing keybindings
type Footer struct {
	*tview.TextView
}

// NewFooter creates a new footer component
func NewFooter() *Footer {
	textView := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft)

	return &Footer{
		TextView: textView,
	}
}

// Update updates the footer with keybinding hints
func (f *Footer) Update(text string) {
	f.SetText(" " + text)
}

