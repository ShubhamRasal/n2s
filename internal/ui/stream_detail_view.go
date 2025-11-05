package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/shubhamrasal/n2s/internal/models"
	"github.com/shubhamrasal/n2s/internal/ui/components"
)

// StreamDetailView displays stream details and consumers
type StreamDetailView struct {
	ui          *UIManager
	flex        *tview.Flex
	infoView    *tview.TextView
	consumerTable *tview.Table
	streamName  string
	stream      *models.Stream
	consumers   []*models.Consumer
}

// NewStreamDetailView creates a new stream detail view
func NewStreamDetailView(ui *UIManager) *StreamDetailView {
	view := &StreamDetailView{
		ui:        ui,
		consumers: make([]*models.Consumer, 0),
	}

	// Info view
	view.infoView = tview.NewTextView().
		SetDynamicColors(true).
		SetWordWrap(true)
	view.infoView.SetBorder(true).
		SetTitle(" Stream Info ").
		SetTitleAlign(tview.AlignCenter)

	// Consumer table
	view.consumerTable = tview.NewTable().
		SetBorders(false).
		SetSelectable(true, false).
		SetFixed(1, 0)
	view.consumerTable.SetBorder(true).
		SetTitle(" Consumers ").
		SetTitleAlign(tview.AlignCenter)

	// Layout
	view.flex = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(view.infoView, 7, 0, false).
		AddItem(view.consumerTable, 0, 1, true)

	view.setupKeybindings()
	view.setupConsumerHeaders()

	return view
}

func (v *StreamDetailView) setupConsumerHeaders() {
	headers := []string{"NAME", "PENDING", "ACK PENDING", "REDELIVERED"}
	for i, header := range headers {
		cell := tview.NewTableCell(header).
			SetTextColor(tcell.ColorYellow).
			SetAlign(tview.AlignLeft).
			SetSelectable(false)
		v.consumerTable.SetCell(0, i, cell)
	}
}

func (v *StreamDetailView) setupKeybindings() {
	v.consumerTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEnter:
			v.onEnter()
			return nil
		case tcell.KeyEsc:
			v.ui.ShowStreamList()
			return nil
		case tcell.KeyRune:
			switch event.Rune() {
			case 'd':
				v.ui.ShowDescribe(v.streamName)
				return nil
			case 'x':
				v.deleteConsumer()
				return nil
			case 'r':
				v.Refresh()
				return nil
			case 'm':
				v.ui.ShowMessages(v.streamName)
				return nil
			case 'e':
				// Check if on consumer row or stream
				row, _ := v.consumerTable.GetSelection()
				if row > 0 && row <= len(v.consumers) {
					// Edit selected consumer
					v.editConsumer()
				} else {
					// Edit stream
					v.ui.ShowStreamEdit(v.streamName)
				}
				return nil
			}
		}
		return event
	})
}

// SetStream sets the stream to display
func (v *StreamDetailView) SetStream(streamName string) {
	v.streamName = streamName
	v.flex.SetTitle(fmt.Sprintf(" Stream: %s ", streamName))
	v.Refresh()
}

// Refresh updates the stream details and consumers
func (v *StreamDetailView) Refresh() {
	if v.streamName == "" {
		return
	}

	// Get stream info
	stream, err := v.ui.client.GetStreamInfo(v.streamName)
	if err != nil {
		v.ui.ShowError(fmt.Sprintf("Failed to get stream info: %v", err))
		return
	}
	v.stream = stream

	// Get consumers
	consumers, err := v.ui.client.ListConsumers(v.streamName)
	if err != nil {
		v.ui.ShowError(fmt.Sprintf("Failed to list consumers: %v", err))
		return
	}
	v.consumers = consumers

	v.updateInfo()
	v.updateConsumerTable()
}

func (v *StreamDetailView) updateInfo() {
	if v.stream == nil {
		return
	}

	info := fmt.Sprintf(
		"[yellow]Subjects:[white] %s\n"+
		"[yellow]Storage:[white] %s    [yellow]Retention:[white] %s    [yellow]Replicas:[white] %d\n"+
		"[yellow]Messages:[white] %s    [yellow]Bytes:[white] %s    [yellow]First Seq:[white] %d    [yellow]Last Seq:[white] %d\n"+
		"[yellow]Max Age:[white] %s    [yellow]Max Msgs:[white] %s    [yellow]Max Bytes:[white] %s",
		strings.Join(v.stream.Subjects, ", "),
		v.stream.Config.Storage,
		v.stream.Config.Retention,
		v.stream.Config.Replicas,
		formatNumber(v.stream.State.Messages),
		formatBytes(v.stream.State.Bytes),
		v.stream.State.FirstSeq,
		v.stream.State.LastSeq,
		formatDuration(v.stream.Config.MaxAge),
		formatNumber(uint64(v.stream.Config.MaxMessages)),
		formatBytes(uint64(v.stream.Config.MaxBytes)),
	)

	v.infoView.SetText(info)
}

func (v *StreamDetailView) updateConsumerTable() {
	// Clear existing rows (keep header)
	for row := v.consumerTable.GetRowCount() - 1; row > 0; row-- {
		v.consumerTable.RemoveRow(row)
	}

	// Add consumer rows
	for i, consumer := range v.consumers {
		row := i + 1
		
		v.consumerTable.SetCell(row, 0, tview.NewTableCell(consumer.Name))
		v.consumerTable.SetCell(row, 1, tview.NewTableCell(fmt.Sprintf("%d", consumer.NumPending)))
		v.consumerTable.SetCell(row, 2, tview.NewTableCell(fmt.Sprintf("%d", consumer.NumAckPending)))
		v.consumerTable.SetCell(row, 3, tview.NewTableCell(fmt.Sprintf("%d", consumer.NumRedelivered)))
	}

	v.ui.footer.Update("Enter: Consumer  d: Describe  e: Edit  m: Messages  x: Delete  r: Refresh  Esc: Back")
}

func (v *StreamDetailView) onEnter() {
	row, _ := v.consumerTable.GetSelection()
	if row > 0 && row <= len(v.consumers) {
		consumer := v.consumers[row-1]
		v.ui.ShowConsumerDetail(v.streamName, consumer.Name)
	}
}

func (v *StreamDetailView) editConsumer() {
	row, _ := v.consumerTable.GetSelection()
	if row > 0 && row <= len(v.consumers) {
		consumer := v.consumers[row-1]
		v.ui.ShowConsumerEdit(v.streamName, consumer.Name)
	}
}

func (v *StreamDetailView) deleteConsumer() {
	if v.ui.readOnly {
		v.ui.ShowError("Cannot delete in read-only mode")
		return
	}

	row, _ := v.consumerTable.GetSelection()
	if row > 0 && row <= len(v.consumers) {
		consumer := v.consumers[row-1]
		
		modal := components.ConfirmModal(
			fmt.Sprintf("Delete consumer '%s'?", consumer.Name),
			func() {
				v.ui.CloseModal()
				if err := v.ui.client.DeleteConsumer(v.streamName, consumer.Name); err != nil {
					v.ui.ShowError(fmt.Sprintf("Failed to delete: %v", err))
				} else {
					v.Refresh()
				}
			},
			func() {
				v.ui.CloseModal()
			},
		)
		
		v.ui.ShowModal(modal)
	}
}

// GetPrimitive returns the primitive for this view
func (v *StreamDetailView) GetPrimitive() tview.Primitive {
	return v.flex
}

func formatDuration(d time.Duration) string {
	if d == 0 {
		return "unlimited"
	}
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	return fmt.Sprintf("%dh", int(d.Hours()))
}

