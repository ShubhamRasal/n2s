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

// ConsumerEditView provides consumer configuration editing
type ConsumerEditView struct {
	ui              *UIManager
	mainFlex        *tview.Flex
	form            *tview.Form
	diffView        *tview.TextView
	infoView        *tview.TextView
	streamName      string
	consumerName    string
	currentConsumer *models.Consumer

	// Editable fields
	maxDeliver    string
	maxAckPending string
	maxWaiting    string
	ackWait       string
	maxBatch      string
	maxExpires    string
	maxBytes      string
	sampleFreq    string
}

// NewConsumerEditView creates a new consumer edit view
func NewConsumerEditView(ui *UIManager) *ConsumerEditView {
	view := &ConsumerEditView{
		ui: ui,
	}

	view.buildUI()
	view.setupKeybindings()

	return view
}

func (v *ConsumerEditView) buildUI() {
	// Create form for editing
	v.form = tview.NewForm()
	v.form.SetBorder(true).
		SetTitle(" Edit Consumer Configuration ").
		SetTitleAlign(tview.AlignCenter)

	// Read-only info panel (top)
	v.infoView = tview.NewTextView().
		SetDynamicColors(true).
		SetWordWrap(true)
	v.infoView.SetBorder(true).
		SetTitle(" Read-Only Fields ").
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

	// Right column: info + diff
	rightColumn := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(v.infoView, 8, 0, false).
		AddItem(v.diffView, 0, 1, false)

	// Layout: form on left, info+diff on right
	v.mainFlex = tview.NewFlex().
		SetDirection(tview.FlexColumn).
		AddItem(v.form, 0, 1, true).
		AddItem(rightColumn, 0, 1, false)
}

func (v *ConsumerEditView) setupKeybindings() {
	v.mainFlex.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			v.ui.ShowStreamDetail(v.streamName)
			return nil
		}
		return event
	})
}

// SetConsumer loads the consumer for editing
func (v *ConsumerEditView) SetConsumer(streamName, consumerName string) {
	v.streamName = streamName
	v.consumerName = consumerName

	// Fetch current consumer config
	consumer, err := v.ui.client.GetConsumerInfo(streamName, consumerName)
	if err != nil {
		v.ui.ShowError(fmt.Sprintf("Failed to load consumer: %v", err))
		return
	}

	v.currentConsumer = consumer
	v.loadCurrentValues()
	v.buildForm()
	v.updateReadOnlyInfo()
}

func (v *ConsumerEditView) loadCurrentValues() {
	// Load current consumer config into form fields
	v.maxDeliver = fmt.Sprintf("%d", v.currentConsumer.Config.MaxDeliver)
	v.maxAckPending = fmt.Sprintf("%d", v.currentConsumer.Config.MaxAckPending)
	v.ackWait = v.currentConsumer.Config.AckWait.String()

	// These might not be available in all versions
	v.maxWaiting = "512" // Default if not available
	v.maxBatch = "0"     // 0 = no limit
	v.maxExpires = "0"
	v.maxBytes = "0"
	v.sampleFreq = "0" // 0 = no sampling
}

func (v *ConsumerEditView) updateReadOnlyInfo() {
	info := fmt.Sprintf(
		"[gray]Name:[white] %s\n"+
			"[gray]Deliver Policy:[white] %s\n"+
			"[gray]Ack Policy:[white] %s\n"+
			"[gray]Replay Policy:[white] %s\n"+
			"[gray]Filter Subject:[white] %s\n"+
			"[gray]Durable:[white] %s",
		v.currentConsumer.Name,
		v.currentConsumer.Config.DeliverPolicy,
		v.currentConsumer.Config.AckPolicy,
		v.currentConsumer.Config.ReplayPolicy,
		v.currentConsumer.Config.FilterSubject,
		v.currentConsumer.Config.Durable,
	)
	v.infoView.SetText(info)
}

func (v *ConsumerEditView) buildForm() {
	v.form.Clear(true)

	v.form.AddTextView("", "[yellow]Editable Fields:[white]", 0, 1, false, false)

	// Max Deliver
	v.form.AddInputField("Max Deliver", v.maxDeliver, 20, nil, func(text string) {
		v.maxDeliver = text
	})

	// Max Ack Pending
	v.form.AddInputField("Max Ack Pending", v.maxAckPending, 20, nil, func(text string) {
		v.maxAckPending = text
	})

	// Ack Wait duration
	v.form.AddInputField("Ack Wait", v.ackWait, 20, nil, func(text string) {
		v.ackWait = text
	})

	// Max Waiting Pulls
	v.form.AddInputField("Max Waiting Pulls", v.maxWaiting, 20, nil, func(text string) {
		v.maxWaiting = text
	})

	// Sample Frequency (0-100)
	v.form.AddInputField("Sample % (0-100)", v.sampleFreq, 20, nil, func(text string) {
		v.sampleFreq = text
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

func (v *ConsumerEditView) previewChanges() {
	var diff strings.Builder

	diff.WriteString("[yellow]Consumer Configuration Changes[white]\n\n")
	diff.WriteString(fmt.Sprintf("Consumer: [cyan]%s[white]\n", v.consumerName))
	diff.WriteString(fmt.Sprintf("Stream: [cyan]%s[white]\n\n", v.streamName))

	hasChanges := false

	// Max Deliver
	newMaxDeliver, _ := strconv.Atoi(v.maxDeliver)
	if newMaxDeliver != v.currentConsumer.Config.MaxDeliver {
		diff.WriteString("Max Deliver:\n")
		diff.WriteString(fmt.Sprintf("  [red]- %d[white]\n", v.currentConsumer.Config.MaxDeliver))
		diff.WriteString(fmt.Sprintf("  [green]+ %d[white]\n\n", newMaxDeliver))
		hasChanges = true
	}

	// Max Ack Pending
	newMaxAckPending, _ := strconv.Atoi(v.maxAckPending)
	if newMaxAckPending != v.currentConsumer.Config.MaxAckPending {
		diff.WriteString("Max Ack Pending:\n")
		diff.WriteString(fmt.Sprintf("  [red]- %d[white]\n", v.currentConsumer.Config.MaxAckPending))
		diff.WriteString(fmt.Sprintf("  [green]+ %d[white]\n\n", newMaxAckPending))
		hasChanges = true
	}

	// Ack Wait
	newAckWait, _ := time.ParseDuration(v.ackWait)
	if newAckWait != v.currentConsumer.Config.AckWait {
		diff.WriteString("Ack Wait:\n")
		diff.WriteString(fmt.Sprintf("  [red]- %s[white]\n", v.currentConsumer.Config.AckWait))
		diff.WriteString(fmt.Sprintf("  [green]+ %s[white]\n\n", newAckWait))
		hasChanges = true
	}

	if !hasChanges {
		diff.WriteString("[gray]No changes detected[white]")
	}

	v.diffView.SetText(diff.String())
	v.diffView.ScrollToBeginning()
}

func (v *ConsumerEditView) applyChanges() {
	if v.ui.readOnly {
		v.ui.ShowError("Cannot edit consumer in read-only mode")
		return
	}

	// Show confirmation
	modal := components.ConfirmModal(
		"Apply changes to consumer configuration?",
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

func (v *ConsumerEditView) performUpdate() {
	// Parse new values
	newMaxDeliver, _ := strconv.Atoi(v.maxDeliver)
	newMaxAckPending, _ := strconv.Atoi(v.maxAckPending)
	newAckWait, _ := time.ParseDuration(v.ackWait)

	// Update consumer via NATS
	err := v.ui.client.UpdateConsumer(v.streamName, v.consumerName, newMaxDeliver, newMaxAckPending, newAckWait)

	if err != nil {
		// Show error with retry option
		v.showUpdateError(err)
		return
	}

	// Show success and go back
	modal := components.InfoModal("Consumer Updated",
		fmt.Sprintf("Consumer '%s' configuration updated successfully!", v.consumerName),
		func() {
			v.ui.CloseModal()
			v.ui.ShowStreamDetail(v.streamName)
		})
	v.ui.ShowModal(modal)
}

func (v *ConsumerEditView) showUpdateError(err error) {
	// Create custom error modal with retry option
	errorMsg := err.Error()

	// Check if it's a "field cannot be changed" error
	cannotChange := strings.Contains(errorMsg, "cannot be changed") ||
		strings.Contains(errorMsg, "immutable") ||
		strings.Contains(errorMsg, "not supported")

	var message string
	if cannotChange {
		message = fmt.Sprintf(
			"[red]Update Failed[white]\n\n%s\n\n"+
				"[yellow]NATS does not allow changing this field after creation.[white]\n\n"+
				"To change immutable fields, you must:\n"+
				"1. Delete this consumer\n"+
				"2. Create a new one with desired settings",
			errorMsg)
	} else {
		message = fmt.Sprintf(
			"[red]Update Failed[white]\n\n%s\n\n"+
				"Check your values and try again.",
			errorMsg)
	}

	// Create form with options
	errorForm := tview.NewForm()
	errorForm.AddTextView("", message, 0, 5, true, false)
	errorForm.AddButton("[ Try Again ]", func() {
		v.ui.CloseModal()
		// Keep current edits, user can fix them
	})
	errorForm.AddButton("[ Reset to Current ]", func() {
		v.ui.CloseModal()
		// Reload original values
		v.loadCurrentValues()
		v.buildForm()
		v.diffView.SetText("[gray]Values reset to current configuration[white]")
	})
	errorForm.AddButton("[ Cancel ]", func() {
		v.ui.CloseModal()
		v.ui.ShowStreamDetail(v.streamName)
	})

	errorForm.SetBorder(true).
		SetTitle(" Error ").
		SetTitleAlign(tview.AlignCenter)

	// Center the form
	centered := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(errorForm, 18, 1, true).
			AddItem(nil, 0, 1, false), 80, 1, true).
		AddItem(nil, 0, 1, false)

	v.ui.ShowModal(centered)
}

// Show shows the edit view
func (v *ConsumerEditView) Show() {
	v.ui.currentPage = "consumer-edit"
	v.ui.pages.SwitchToPage("consumer-edit")
	v.ui.app.SetFocus(v.form)
	v.ui.footer.Update("Tab: Navigate  Enter: Select  Esc: Cancel")
}

// GetPrimitive returns the primitive for this view
func (v *ConsumerEditView) GetPrimitive() tview.Primitive {
	return v.mainFlex
}
