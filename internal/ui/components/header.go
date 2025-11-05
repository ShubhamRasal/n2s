package components

import (
	"fmt"

	"github.com/rivo/tview"
)

// Header represents the application header component
type Header struct {
	*tview.TextView
}

// NewHeader creates a new header component
func NewHeader() *Header {
	textView := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft)

	return &Header{
		TextView: textView,
	}
}

// Update updates the header with connection and context info
func (h *Header) Update(contextName, status string, readOnly bool) {
	readOnlyIndicator := ""
	if readOnly {
		readOnlyIndicator = " [yellow][READ-ONLY][white]"
	}

	header := fmt.Sprintf("[yellow]N2S[white] - NATS JetStream TUI          Context: [cyan]%s[white]      %s%s",
		contextName,
		status,
		readOnlyIndicator,
	)
	h.SetText(header)
}

// UpdateStatus updates only the status portion
func (h *Header) UpdateStatus(connected bool) string {
	if connected {
		return "[green]●[white] Connected"
	}
	return "[red]●[white] Disconnected"
}

