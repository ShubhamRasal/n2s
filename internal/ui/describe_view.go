package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/shubhamrasal/n2s/internal/models"
)

// DescribeView displays stream description and details
type DescribeView struct {
	ui         *UIManager
	flex       *tview.Flex
	textView   *tview.TextView
	streamName string
	stream     *models.Stream
	consumers  []*models.Consumer
}

// NewDescribeView creates a new describe view
func NewDescribeView(ui *UIManager) *DescribeView {
	view := &DescribeView{
		ui: ui,
	}

	view.textView = tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetWordWrap(true)

	view.textView.SetBorder(true).
		SetTitle(" Stream Description ").
		SetTitleAlign(tview.AlignCenter)

	view.flex = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(view.textView, 0, 1, true)

	view.setupKeybindings()

	return view
}

func (v *DescribeView) setupKeybindings() {
	v.flex.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEsc:
			v.ui.ShowStreamDetail(v.streamName)
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

// SetStream sets the stream to describe
func (v *DescribeView) SetStream(streamName string) {
	v.streamName = streamName
	v.flex.SetTitle(fmt.Sprintf(" Describe: %s ", streamName))
	v.Refresh()
}

// Refresh updates the description
func (v *DescribeView) Refresh() {
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
		v.ui.ShowError(fmt.Sprintf("Failed to get consumers: %v", err))
		return
	}
	v.consumers = consumers

	v.updateMetrics()
}

func (v *DescribeView) updateMetrics() {
	if v.stream == nil {
		return
	}

	var output strings.Builder

	// Stream Overview
	output.WriteString("[yellow]═══ STREAM OVERVIEW ═══[white]\n\n")
	output.WriteString(fmt.Sprintf("[cyan]Name:[white]         %s\n", v.stream.Name))
	output.WriteString(fmt.Sprintf("[cyan]Storage:[white]      %s\n", v.stream.Config.Storage))
	output.WriteString(fmt.Sprintf("[cyan]Retention:[white]    %s\n", v.stream.Config.Retention))
	output.WriteString(fmt.Sprintf("[cyan]Replicas:[white]     %d\n\n", v.stream.Config.Replicas))

	// Message Stats
	output.WriteString("[yellow]═══ MESSAGE STATISTICS ═══[white]\n\n")
	output.WriteString(fmt.Sprintf("[cyan]Total Messages:[white]  %s\n", formatNumber(v.stream.State.Messages)))
	output.WriteString(fmt.Sprintf("[cyan]Total Bytes:[white]     %s\n", formatBytes(v.stream.State.Bytes)))
	output.WriteString(fmt.Sprintf("[cyan]First Seq:[white]       %d\n", v.stream.State.FirstSeq))
	output.WriteString(fmt.Sprintf("[cyan]Last Seq:[white]        %d\n", v.stream.State.LastSeq))
	output.WriteString(fmt.Sprintf("[cyan]Deleted:[white]         %d\n\n", v.stream.State.NumDeleted))

	// Visual message count bar
	output.WriteString("[yellow]Message Count:[white]\n")
	output.WriteString(v.createBar(v.stream.State.Messages, uint64(v.stream.Config.MaxMessages)))
	output.WriteString("\n\n")

	// Storage usage bar
	output.WriteString("[yellow]Storage Usage:[white]\n")
	output.WriteString(v.createBar(v.stream.State.Bytes, uint64(v.stream.Config.MaxBytes)))
	output.WriteString("\n\n")

	// Limits
	output.WriteString("[yellow]═══ LIMITS ═══[white]\n\n")
	output.WriteString(fmt.Sprintf("[cyan]Max Age:[white]         %s\n", formatDuration(v.stream.Config.MaxAge)))
	output.WriteString(fmt.Sprintf("[cyan]Max Messages:[white]    %s\n", formatNumber(uint64(v.stream.Config.MaxMessages))))
	output.WriteString(fmt.Sprintf("[cyan]Max Bytes:[white]       %s\n", formatBytes(uint64(v.stream.Config.MaxBytes))))
	output.WriteString(fmt.Sprintf("[cyan]Max Msg Size:[white]    %s\n\n", formatBytes(uint64(v.stream.Config.MaxMsgSize))))

	// Consumer Stats
	output.WriteString("[yellow]═══ CONSUMER STATISTICS ═══[white]\n\n")
	output.WriteString(fmt.Sprintf("[cyan]Total Consumers:[white] %d\n\n", len(v.consumers)))

	if len(v.consumers) > 0 {
		// Consumer table
		output.WriteString(fmt.Sprintf("%-30s %10s %10s %12s\n",
			"[yellow]CONSUMER[white]",
			"[yellow]PENDING[white]",
			"[yellow]ACK PEND[white]",
			"[yellow]REDELIVERED[white]"))
		output.WriteString(strings.Repeat("─", 70) + "\n")

		for _, consumer := range v.consumers {
			output.WriteString(fmt.Sprintf("%-30s %10d %10d %12d\n",
				consumer.Name,
				consumer.NumPending,
				consumer.NumAckPending,
				consumer.NumRedelivered))
		}
		output.WriteString("\n")

		// Total pending across all consumers
		totalPending := uint64(0)
		totalAckPending := uint64(0)
		for _, c := range v.consumers {
			totalPending += c.NumPending
			totalAckPending += c.NumAckPending
		}
		output.WriteString(fmt.Sprintf("\n[cyan]Total Pending:[white]     %d\n", totalPending))
		output.WriteString(fmt.Sprintf("[cyan]Total Ack Pending:[white] %d\n", totalAckPending))
	}

	// Time info
	output.WriteString(fmt.Sprintf("\n\n[yellow]═══ TIMESTAMPS ═══[white]\n\n"))
	output.WriteString(fmt.Sprintf("[cyan]First Message:[white] %s\n", v.stream.State.FirstTime.Format("2006-01-02 15:04:05")))
	output.WriteString(fmt.Sprintf("[cyan]Last Message:[white]  %s\n", v.stream.State.LastTime.Format("2006-01-02 15:04:05")))
	
	// Age of stream
	age := time.Since(v.stream.State.FirstTime)
	output.WriteString(fmt.Sprintf("[cyan]Stream Age:[white]    %s\n", formatDuration(age)))

	v.textView.SetText(output.String())
	v.ui.footer.Update("r: Refresh  Esc: Back to Stream")
}

// createBar creates a visual progress bar
func (v *DescribeView) createBar(current, max uint64) string {
	if max == 0 {
		return "[green][████████████████████████████████████████] unlimited[white]"
	}

	percentage := float64(current) / float64(max) * 100
	barWidth := 40
	filled := int(percentage * float64(barWidth) / 100)

	if filled > barWidth {
		filled = barWidth
	}

	color := "green"
	if percentage > 80 {
		color = "red"
	} else if percentage > 60 {
		color = "yellow"
	}

	bar := fmt.Sprintf("[%s][%s%s][white] %.1f%% (%s / %s)",
		color,
		strings.Repeat("█", filled),
		strings.Repeat("░", barWidth-filled),
		percentage,
		formatNumber(current),
		formatNumber(max))

	return bar
}

// GetPrimitive returns the primitive for this view
func (v *DescribeView) GetPrimitive() tview.Primitive {
	return v.flex
}

