package ui

import (
	"fmt"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/shubhamrasal/n2s/internal/models"
)

// MessageView displays messages from a stream
type MessageView struct {
	ui            *UIManager
	flex          *tview.Flex
	messageTable  *tview.Table
	detailView    *tview.TextView
	streamName    string
	messages      []*models.Message
	selectedMsg   *models.MessageDetail
	loading       bool
	loadingStart  time.Time
	focusOnDetail bool
	stopLoading   chan bool
}

// NewMessageView creates a new message view
func NewMessageView(ui *UIManager) *MessageView {
	view := &MessageView{
		ui:       ui,
		messages: make([]*models.Message, 0),
	}

	// Message table
	view.messageTable = tview.NewTable().
		SetBorders(false).
		SetSelectable(true, false).
		SetFixed(1, 0)
	view.messageTable.SetBorder(true).
		SetTitle(" Messages ").
		SetTitleAlign(tview.AlignCenter)

	// Detail view
	view.detailView = tview.NewTextView().
		SetDynamicColors(true).
		SetWordWrap(true).
		SetScrollable(true)
	view.detailView.SetBorder(true).
		SetTitle(" Message Detail ").
		SetTitleAlign(tview.AlignCenter)

	// Layout
	view.flex = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(view.messageTable, 0, 1, true).
		AddItem(view.detailView, 0, 1, false)

	view.setupKeybindings()
	view.setupHeaders()

	return view
}

func (view *MessageView) setupHeaders() {
	headers := []string{"SEQ", "SUBJECT", "TIME", "SIZE"}
	for i, header := range headers {
		cell := tview.NewTableCell(header).
			SetTextColor(tcell.ColorYellow).
			SetAlign(tview.AlignLeft).
			SetSelectable(false)
		view.messageTable.SetCell(0, i, cell)
	}
}

func (v *MessageView) setupKeybindings() {
	v.messageTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEnter:
			v.onEnter()
			return nil
		case tcell.KeyEsc:
			v.clearAndGoBack()
			return nil
		case tcell.KeyTab:
			v.switchFocus()
			return nil
		case tcell.KeyRune:
			switch event.Rune() {
			case 'r':
				v.Refresh()
				return nil
			}
		}
		return event
	})

	v.detailView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEsc:
			v.clearAndGoBack()
			return nil
		case tcell.KeyTab:
			v.switchFocus()
			return nil
		case tcell.KeyRune:
			switch event.Rune() {
			case 'r':
				v.Refresh()
				return nil
			}
		}
		return event
	})
}

func (v *MessageView) clearAndGoBack() {
	// Clear message detail
	v.selectedMsg = nil
	v.messages = []*models.Message{}
	v.detailView.Clear()
	v.detailView.SetText("[gray]Select a message to view details[white]")
	
	// Clear message table
	v.messageTable.Clear()
	v.setupHeaders()
	
	// Go back to stream list
	v.ui.ShowStreamList()
}

// SetStream sets the stream to display messages from
func (v *MessageView) SetStream(streamName string) {
	v.streamName = streamName
	v.flex.SetTitle(fmt.Sprintf(" Messages: %s ", streamName))
	
	// Clear old message detail when switching streams
	v.selectedMsg = nil
	v.detailView.Clear()
	v.detailView.SetText("[gray]Select a message to view details[white]")
	
	v.Refresh()
}

// Refresh updates the message list
func (v *MessageView) Refresh() {
	if v.streamName == "" {
		return
	}

	// Prevent concurrent loads
	if v.loading {
		return
	}
	v.loading = true
	v.loadingStart = time.Now()
	v.stopLoading = make(chan bool, 1)

	// Show animated loading indicator
	v.showLoadingAnimation()

	// Fetch messages in background
	go func() {
		messages, err := v.ui.client.ListMessages(v.streamName, 20)
		
		// Update UI on main thread
		v.ui.app.QueueUpdateDraw(func() {
			v.stopLoadingAnimation()
			
			if err != nil {
				v.ui.ShowError(fmt.Sprintf("Failed to list messages: %v", err))
				v.messageTable.Clear()
				v.setupHeaders()
				return
			}

			v.messages = messages
			v.updateTable()
		})
	}()
}

func (v *MessageView) stopLoadingAnimation() {
	v.loading = false
	if v.stopLoading != nil {
		select {
		case v.stopLoading <- true:
		default:
		}
	}
}

func (v *MessageView) showLoadingAnimation() {
	// Clear and show simple loading message
	v.messageTable.Clear()
	v.setupHeaders()
	
	loadingMsg := "[yellow]⏳ Loading messages... (max 60s)[white]"
	v.messageTable.SetCell(1, 0, tview.NewTableCell(loadingMsg).SetAlign(tview.AlignCenter))
	v.ui.footer.Update("[yellow]Loading messages from NATS...[white] Please wait")
}

func (v *MessageView) switchFocus() {
	v.focusOnDetail = !v.focusOnDetail
	
	if v.focusOnDetail {
		// Focus on detail view for scrolling
		v.messageTable.SetBorderColor(tcell.ColorGray)
		v.detailView.SetBorderColor(tcell.ColorGreen)
		v.ui.app.SetFocus(v.detailView)
		v.ui.footer.Update("↑/↓/PgUp/PgDn: Scroll  Tab: Switch pane  r: Refresh  Esc: Back")
	} else {
		// Focus on message table
		v.messageTable.SetBorderColor(tcell.ColorGreen)
		v.detailView.SetBorderColor(tcell.ColorGray)
		v.ui.app.SetFocus(v.messageTable)
		v.ui.footer.Update(fmt.Sprintf("Enter: View Detail  Tab: Switch pane  r: Refresh  Esc: Back  [Showing last %d messages]", len(v.messages)))
	}
}

func (v *MessageView) updateTable() {
	// Clear existing rows (keep header)
	for row := v.messageTable.GetRowCount() - 1; row > 0; row-- {
		v.messageTable.RemoveRow(row)
	}

	// Clear detail panel when refreshing
	v.selectedMsg = nil
	v.detailView.Clear()
	v.detailView.SetText("[gray]Select a message to view details[white]")

	// Add message rows (reverse order - newest first)
	for i := len(v.messages) - 1; i >= 0; i-- {
		msg := v.messages[i]
		row := len(v.messages) - i
		
		// Format timestamp
		timeStr := formatTime(msg.Timestamp)
		
		// Truncate subject if too long
		subject := msg.Subject
		if len(subject) > 40 {
			subject = subject[:37] + "..."
		}
		
		v.messageTable.SetCell(row, 0, tview.NewTableCell(fmt.Sprintf("%d", msg.Sequence)))
		v.messageTable.SetCell(row, 1, tview.NewTableCell(subject))
		v.messageTable.SetCell(row, 2, tview.NewTableCell(timeStr))
		v.messageTable.SetCell(row, 3, tview.NewTableCell(formatBytes(uint64(msg.Size))))
	}

	// Reset focus to message table
	v.focusOnDetail = false
	v.messageTable.SetBorderColor(tcell.ColorGreen)
	v.detailView.SetBorderColor(tcell.ColorGray)
	v.ui.footer.Update(fmt.Sprintf("Enter: View Detail  Tab: Switch pane  r: Refresh  Esc: Back  [Showing last %d messages]", len(v.messages)))
}

func (v *MessageView) onEnter() {
	row, _ := v.messageTable.GetSelection()
	if row > 0 && row <= len(v.messages) {
		// Messages are in reverse order in the table
		idx := len(v.messages) - row
		msg := v.messages[idx]
		
		// Get full message detail
		detail, err := v.ui.client.GetMessageDetail(v.streamName, msg.Sequence)
		if err != nil {
			v.ui.ShowError(fmt.Sprintf("Failed to get message: %v", err))
			return
		}
		
		v.selectedMsg = detail
		v.updateDetail()
	}
}

func (v *MessageView) updateDetail() {
	if v.selectedMsg == nil {
		v.detailView.SetText("Select a message to view details")
		return
	}

	// Format headers
	headersText := ""
	if len(v.selectedMsg.Headers) > 0 {
		headersText = "[yellow]Headers:[white]\n"
		for k, vals := range v.selectedMsg.Headers {
			for _, val := range vals {
				headersText += fmt.Sprintf("  %s: %s\n", k, val)
			}
		}
		headersText += "\n"
	}

	detail := fmt.Sprintf(
		"[yellow]Sequence:[white] %d\n"+
		"[yellow]Subject:[white] %s\n"+
		"[yellow]Time:[white] %s\n"+
		"[yellow]Size:[white] %s\n\n"+
		"%s"+
		"[yellow]Payload:[white]\n%s",
		v.selectedMsg.Sequence,
		v.selectedMsg.Subject,
		v.selectedMsg.Timestamp.Format("2006-01-02 15:04:05"),
		formatBytes(uint64(v.selectedMsg.Size)),
		headersText,
		v.selectedMsg.Payload,
	)

	v.detailView.SetText(detail)
}

// GetPrimitive returns the primitive for this view
func (v *MessageView) GetPrimitive() tview.Primitive {
	return v.flex
}

func formatTime(t time.Time) string {
	now := time.Now()
	diff := now.Sub(t)

	if diff < time.Minute {
		return fmt.Sprintf("%ds ago", int(diff.Seconds()))
	} else if diff < time.Hour {
		return fmt.Sprintf("%dm ago", int(diff.Minutes()))
	} else if diff < 24*time.Hour {
		return fmt.Sprintf("%dh ago", int(diff.Hours()))
	} else if diff < 7*24*time.Hour {
		return fmt.Sprintf("%dd ago", int(diff.Hours()/24))
	}
	return t.Format("2006-01-02")
}

