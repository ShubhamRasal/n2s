package ui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/shubhamrasal/n2s/internal/models"
	"github.com/shubhamrasal/n2s/internal/ui/components"
)

// ConsumerDetailView displays detailed consumer information
type ConsumerDetailView struct {
	ui           *UIManager
	flex         *tview.Flex
	infoView     *tview.TextView
	metricsView  *tview.TextView
	streamName   string
	consumerName string
	consumer     *models.Consumer
}

// NewConsumerDetailView creates a new consumer detail view
func NewConsumerDetailView(ui *UIManager) *ConsumerDetailView {
	view := &ConsumerDetailView{
		ui: ui,
	}

	// Info view
	view.infoView = tview.NewTextView().
		SetDynamicColors(true).
		SetWordWrap(true)
	view.infoView.SetBorder(true).
		SetTitle(" Consumer Info ").
		SetTitleAlign(tview.AlignCenter)

	// Metrics view
	view.metricsView = tview.NewTextView().
		SetDynamicColors(true).
		SetWordWrap(true)
	view.metricsView.SetBorder(true).
		SetTitle(" Metrics (Live) ").
		SetTitleAlign(tview.AlignCenter)

	// Layout
	view.flex = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(view.infoView, 6, 0, false).
		AddItem(view.metricsView, 0, 1, true)

	view.setupKeybindings()

	return view
}

func (v *ConsumerDetailView) setupKeybindings() {
	v.flex.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEsc:
			v.ui.ShowStreamDetail(v.streamName)
			return nil
		case tcell.KeyRune:
			switch event.Rune() {
			case 'd':
				v.deleteConsumer()
				return nil
			case 'r':
				v.Refresh()
				return nil
			}
		}
		return event
	})
}

// SetConsumer sets the consumer to display
func (v *ConsumerDetailView) SetConsumer(streamName, consumerName string) {
	v.streamName = streamName
	v.consumerName = consumerName
	v.flex.SetTitle(fmt.Sprintf(" Consumer: %s (Stream: %s) ", consumerName, streamName))
	v.Refresh()
}

// Refresh updates the consumer details
func (v *ConsumerDetailView) Refresh() {
	if v.streamName == "" || v.consumerName == "" {
		return
	}

	// Get consumer info
	consumer, err := v.ui.client.GetConsumerInfo(v.streamName, v.consumerName)
	if err != nil {
		v.ui.ShowError(fmt.Sprintf("Failed to get consumer info: %v", err))
		return
	}
	v.consumer = consumer

	v.updateInfo()
	v.updateMetrics()
}

func (v *ConsumerDetailView) updateInfo() {
	if v.consumer == nil {
		return
	}

	info := fmt.Sprintf(
		"[yellow]Deliver Policy:[white] %s    [yellow]Ack Policy:[white] %s    [yellow]Ack Wait:[white] %s\n"+
		"[yellow]Max Deliver:[white] %d    [yellow]Replay Policy:[white] %s\n"+
		"[yellow]Filter:[white] %s\n"+
		"[yellow]Durable:[white] %s",
		v.consumer.Config.DeliverPolicy,
		v.consumer.Config.AckPolicy,
		formatDuration(v.consumer.Config.AckWait),
		v.consumer.Config.MaxDeliver,
		v.consumer.Config.ReplayPolicy,
		v.consumer.Config.FilterSubject,
		v.consumer.Config.Durable,
	)

	v.infoView.SetText(info)
}

func (v *ConsumerDetailView) updateMetrics() {
	if v.consumer == nil {
		return
	}

	metrics := fmt.Sprintf(
		"[yellow]Delivered:[white] %d    [yellow]Acknowledged:[white] %d\n"+
		"[yellow]Pending:[white] %d    [yellow]Ack Pending:[white] %d\n"+
		"[yellow]Redelivered:[white] %d    [yellow]Waiting:[white] %d\n\n"+
		"[green]Stream Sequence:[white] %d\n"+
		"[green]Consumer Sequence:[white] %d",
		v.consumer.Delivered.Stream,
		v.consumer.AckFloor.Stream,
		v.consumer.NumPending,
		v.consumer.NumAckPending,
		v.consumer.NumRedelivered,
		v.consumer.NumWaiting,
		v.consumer.Delivered.Stream,
		v.consumer.Delivered.Consumer,
	)

	v.metricsView.SetText(metrics)
}

func (v *ConsumerDetailView) deleteConsumer() {
	if v.ui.readOnly {
		v.ui.ShowError("Cannot delete in read-only mode")
		return
	}

	modal := components.ConfirmModal(
		fmt.Sprintf("Delete consumer '%s'?", v.consumerName),
		func() {
			v.ui.CloseModal()
			if err := v.ui.client.DeleteConsumer(v.streamName, v.consumerName); err != nil {
				v.ui.ShowError(fmt.Sprintf("Failed to delete: %v", err))
			} else {
				v.ui.ShowStreamDetail(v.streamName)
			}
		},
		func() {
			v.ui.CloseModal()
		},
	)
	
	v.ui.ShowModal(modal)
}

// GetPrimitive returns the primitive for this view
func (v *ConsumerDetailView) GetPrimitive() tview.Primitive {
	return v.flex
}

