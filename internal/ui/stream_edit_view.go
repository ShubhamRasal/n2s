package ui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/shubhamrasal/n2s/internal/models"
	"github.com/shubhamrasal/n2s/internal/ui/components"
)

// StreamEditView provides stream configuration editing
type StreamEditView struct {
	ui           *UIManager
	mainFlex     *tview.Flex
	form         *tview.Form
	diffView     *tview.TextView
	streamName   string
	currentStream *models.Stream
	
	// Editable fields
	maxMsgs      string
	maxBytes     string
	maxAge       string
	maxMsgSize   string
	retention    string
	discard      string
	compression  string
}

// NewStreamEditView creates a new stream edit view
func NewStreamEditView(ui *UIManager) *StreamEditView {
	view := &StreamEditView{
		ui: ui,
	}

	view.buildUI()
	view.setupKeybindings()

	return view
}

func (v *StreamEditView) buildUI() {
	// Create form for editing
	v.form = tview.NewForm()
	v.form.SetBorder(true).
		SetTitle(" Edit Stream Configuration ").
		SetTitleAlign(tview.AlignCenter)

	// Diff preview panel
	v.diffView = tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetWordWrap(false)
	v.diffView.SetBorder(true).
		SetTitle(" Changes Preview ").
		SetTitleAlign(tview.AlignCenter)
	v.diffView.SetText("[gray]Make changes and click 'Preview' to see diff[white]")

	// Layout: form on left, diff on right
	v.mainFlex = tview.NewFlex().
		SetDirection(tview.FlexColumn).
		AddItem(v.form, 0, 1, true).
		AddItem(v.diffView, 0, 1, false)
}

func (v *StreamEditView) setupKeybindings() {
	v.mainFlex.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			v.ui.ShowStreamDetail(v.streamName)
			return nil
		}
		return event
	})
}

// SetStream loads the stream for editing
func (v *StreamEditView) SetStream(streamName string) {
	v.streamName = streamName
	
	// Fetch current stream config
	stream, err := v.ui.client.GetStreamInfo(streamName)
	if err != nil {
		v.ui.ShowError(fmt.Sprintf("Failed to load stream: %v", err))
		return
	}
	
	v.currentStream = stream
	v.loadCurrentValues()
	v.buildForm()
}

func (v *StreamEditView) loadCurrentValues() {
	// Load current stream config into form fields
	if v.currentStream.Config.MaxMessages == -1 {
		v.maxMsgs = "unlimited"
	} else {
		v.maxMsgs = fmt.Sprintf("%d", v.currentStream.Config.MaxMessages)
	}
	
	if v.currentStream.Config.MaxBytes == -1 {
		v.maxBytes = "unlimited"
	} else {
		v.maxBytes = formatBytesToString(uint64(v.currentStream.Config.MaxBytes))
	}
	
	if v.currentStream.Config.MaxAge == 0 {
		v.maxAge = "unlimited"
	} else {
		v.maxAge = formatDurationToString(v.currentStream.Config.MaxAge)
	}
	
	if v.currentStream.Config.MaxMsgSize == -1 {
		v.maxMsgSize = "unlimited"
	} else {
		v.maxMsgSize = formatBytesToString(uint64(v.currentStream.Config.MaxMsgSize))
	}
	
	v.retention = strings.ToLower(v.currentStream.Config.Retention)
	v.discard = strings.ToLower(v.currentStream.Config.Discard)
}

func (v *StreamEditView) buildForm() {
	v.form.Clear(true)
	
	// Max Messages
	v.form.AddInputField("Max Messages", v.maxMsgs, 20, nil, func(text string) {
		v.maxMsgs = text
	})
	
	// Max Bytes (human readable: 5GB, 100MB, etc)
	v.form.AddInputField("Max Bytes", v.maxBytes, 20, nil, func(text string) {
		v.maxBytes = text
	})
	
	// Max Age (human readable: 24h, 7d, etc)
	v.form.AddInputField("Max Age", v.maxAge, 20, nil, func(text string) {
		v.maxAge = text
	})
	
	// Max Message Size
	v.form.AddInputField("Max Msg Size", v.maxMsgSize, 20, nil, func(text string) {
		v.maxMsgSize = text
	})
	
	// Retention policy dropdown
	retentionOpts := []string{"limits", "interest", "workqueue"}
	retentionIdx := 0
	for i, opt := range retentionOpts {
		if opt == v.retention {
			retentionIdx = i
			break
		}
	}
	v.form.AddDropDown("Retention", retentionOpts, retentionIdx, func(option string, index int) {
		v.retention = option
	})
	
	// Discard policy dropdown
	discardOpts := []string{"old", "new"}
	discardIdx := 0
	for i, opt := range discardOpts {
		if opt == v.discard {
			discardIdx = i
			break
		}
	}
	v.form.AddDropDown("Discard", discardOpts, discardIdx, func(option string, index int) {
		v.discard = option
	})
	
	// Buttons
	v.form.AddButton("[ Preview Changes ]", func() {
		v.previewChanges()
	})
	
	v.form.AddButton("[ Apply Changes ]", func() {
		v.applyChanges()
	})
	
	v.form.AddButton("[ Cancel ]", func() {
		v.ui.ShowStreamDetail(v.streamName)
	})
}

func (v *StreamEditView) previewChanges() {
	var diff strings.Builder
	
	diff.WriteString("[yellow]Stream Configuration Changes[white]\n\n")
	diff.WriteString(fmt.Sprintf("Stream: [cyan]%s[white]\n\n", v.streamName))
	
	// Compare old vs new
	hasChanges := false
	
	// Parse new values
	var newMaxMsgs int64
	if strings.ToLower(v.maxMsgs) == "unlimited" {
		newMaxMsgs = -1
	} else {
		newMaxMsgs, _ = strconv.ParseInt(v.maxMsgs, 10, 64)
	}
	
	// Max Messages
	if newMaxMsgs != v.currentStream.Config.MaxMessages {
		diff.WriteString(fmt.Sprintf("Max Messages:\n"))
		if v.currentStream.Config.MaxMessages == -1 {
			diff.WriteString(fmt.Sprintf("  [red]- unlimited[white]\n"))
		} else {
			diff.WriteString(fmt.Sprintf("  [red]- %d[white]\n", v.currentStream.Config.MaxMessages))
		}
		if newMaxMsgs == -1 {
			diff.WriteString(fmt.Sprintf("  [green]+ unlimited[white]\n\n"))
		} else {
			diff.WriteString(fmt.Sprintf("  [green]+ %d[white]\n\n", newMaxMsgs))
		}
		hasChanges = true
	}
	
	// Max Bytes
	var newMaxBytes int64
	if strings.ToLower(v.maxBytes) == "unlimited" {
		newMaxBytes = -1
	} else {
		newMaxBytes = int64(parseByteString(v.maxBytes))
	}
	
	if newMaxBytes != v.currentStream.Config.MaxBytes {
		diff.WriteString(fmt.Sprintf("Max Bytes:\n"))
		diff.WriteString(fmt.Sprintf("  [red]- %s[white]\n", formatBytes(uint64(v.currentStream.Config.MaxBytes))))
		diff.WriteString(fmt.Sprintf("  [green]+ %s[white]\n\n", formatBytes(uint64(newMaxBytes))))
		hasChanges = true
	}
	
	// Max Age
	var newMaxAge time.Duration
	if strings.ToLower(v.maxAge) == "unlimited" {
		newMaxAge = 0
	} else {
		newMaxAge, _ = time.ParseDuration(v.maxAge)
	}
	
	if newMaxAge != v.currentStream.Config.MaxAge {
		diff.WriteString(fmt.Sprintf("Max Age:\n"))
		diff.WriteString(fmt.Sprintf("  [red]- %s[white]\n", v.currentStream.Config.MaxAge))
		diff.WriteString(fmt.Sprintf("  [green]+ %s[white]\n\n", newMaxAge))
		hasChanges = true
	}
	
	// Max Message Size
	var newMaxMsgSize int32
	if strings.ToLower(v.maxMsgSize) == "unlimited" {
		newMaxMsgSize = -1
	} else {
		newMaxMsgSize = int32(parseByteString(v.maxMsgSize))
	}
	
	if newMaxMsgSize != v.currentStream.Config.MaxMsgSize {
		diff.WriteString(fmt.Sprintf("Max Message Size:\n"))
		diff.WriteString(fmt.Sprintf("  [red]- %s[white]\n", formatBytes(uint64(v.currentStream.Config.MaxMsgSize))))
		diff.WriteString(fmt.Sprintf("  [green]+ %s[white]\n\n", formatBytes(uint64(newMaxMsgSize))))
		hasChanges = true
	}
	
	// Retention (case-insensitive comparison)
	if !strings.EqualFold(v.retention, v.currentStream.Config.Retention) {
		diff.WriteString(fmt.Sprintf("Retention:\n"))
		diff.WriteString(fmt.Sprintf("  [red]- %s[white]\n", v.currentStream.Config.Retention))
		diff.WriteString(fmt.Sprintf("  [green]+ %s[white]\n\n", v.retention))
		hasChanges = true
	}
	
	// Discard (case-insensitive comparison)
	if !strings.EqualFold(v.discard, v.currentStream.Config.Discard) {
		diff.WriteString(fmt.Sprintf("Discard:\n"))
		diff.WriteString(fmt.Sprintf("  [red]- %s[white]\n", v.currentStream.Config.Discard))
		diff.WriteString(fmt.Sprintf("  [green]+ %s[white]\n\n", v.discard))
		hasChanges = true
	}
	
	if !hasChanges {
		diff.WriteString("[gray]No changes detected[white]")
	}
	
	v.diffView.SetText(diff.String())
	v.diffView.ScrollToBeginning()
}

func (v *StreamEditView) applyChanges() {
	if v.ui.readOnly {
		v.ui.ShowError("Cannot edit stream in read-only mode")
		return
	}
	
	// Show confirmation
	modal := components.ConfirmModal(
		"Apply changes to stream configuration?\n\nThis will update the stream settings.",
		func() {
			v.ui.CloseModal()
			v.performUpdate()
		},
		func() {
			v.ui.CloseModal()
		},
	)
	
	v.ui.ShowModal(modal)
}

func (v *StreamEditView) performUpdate() {
	// Parse new values
	var newMaxMsgs int64
	if strings.ToLower(v.maxMsgs) == "unlimited" {
		newMaxMsgs = -1
	} else {
		newMaxMsgs, _ = strconv.ParseInt(v.maxMsgs, 10, 64)
	}
	
	var newMaxBytes int64
	if strings.ToLower(v.maxBytes) == "unlimited" {
		newMaxBytes = -1
	} else {
		newMaxBytes = int64(parseByteString(v.maxBytes))
	}
	
	var newMaxAge time.Duration
	if strings.ToLower(v.maxAge) == "unlimited" {
		newMaxAge = 0
	} else {
		newMaxAge, _ = time.ParseDuration(v.maxAge)
	}
	
	var newMaxMsgSize int32
	if strings.ToLower(v.maxMsgSize) == "unlimited" {
		newMaxMsgSize = -1
	} else {
		newMaxMsgSize = int32(parseByteString(v.maxMsgSize))
	}
	
	// Update stream via NATS
	err := v.ui.client.UpdateStream(v.streamName, newMaxMsgs, newMaxBytes, newMaxAge, newMaxMsgSize, v.retention, v.discard)
	
	if err != nil {
		v.ui.ShowError(fmt.Sprintf("Failed to update stream: %v", err))
		return
	}
	
	// Show success and go back
	modal := components.InfoModal("Stream Updated",
		fmt.Sprintf("Stream '%s' configuration updated successfully!", v.streamName),
		func() {
			v.ui.CloseModal()
			v.ui.ShowStreamDetail(v.streamName)
		})
	v.ui.ShowModal(modal)
}

// Show shows the edit view
func (v *StreamEditView) Show() {
	v.ui.currentPage = "stream-edit"
	v.ui.pages.SwitchToPage("stream-edit")
	v.ui.app.SetFocus(v.form)
	v.ui.footer.Update("Tab: Navigate  Enter: Select  Esc: Cancel")
}

// GetPrimitive returns the primitive for this view
func (v *StreamEditView) GetPrimitive() tview.Primitive {
	return v.mainFlex
}

// Helper functions
func formatBytesToString(bytes uint64) string {
	// Handle unlimited (stored as -1, but passed as uint64 max value)
	if bytes == 0 || bytes > uint64(1<<62) {
		return "unlimited"
	}
	
	if bytes >= 1024*1024*1024 {
		return fmt.Sprintf("%.0fGB", float64(bytes)/(1024*1024*1024))
	} else if bytes >= 1024*1024 {
		return fmt.Sprintf("%.0fMB", float64(bytes)/(1024*1024))
	} else if bytes >= 1024 {
		return fmt.Sprintf("%.0fKB", float64(bytes)/1024)
	}
	return fmt.Sprintf("%d", bytes)
}

func formatDurationToString(d time.Duration) string {
	if d == 0 {
		return "unlimited"
	}
	return d.String()
}

func parseByteString(s string) uint64 {
	s = strings.TrimSpace(s)
	
	// Check for unlimited - return 0, caller will handle
	if strings.ToLower(s) == "unlimited" {
		return 0
	}
	
	s = strings.ToUpper(s)
	
	// Extract number and unit
	var value float64
	var unit string
	fmt.Sscanf(s, "%f%s", &value, &unit)
	
	switch unit {
	case "GB":
		return uint64(value * 1024 * 1024 * 1024)
	case "MB":
		return uint64(value * 1024 * 1024)
	case "KB":
		return uint64(value * 1024)
	default:
		val, _ := strconv.ParseUint(s, 10, 64)
		return val
	}
}

