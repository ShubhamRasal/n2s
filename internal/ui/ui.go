package ui

import (
	"fmt"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/shubhamrasal/n2s/internal/config"
	"github.com/shubhamrasal/n2s/internal/nats"
	"github.com/shubhamrasal/n2s/internal/plugins"
	"github.com/shubhamrasal/n2s/internal/ui/components"
)

// UIManager manages the application UI
type UIManager struct {
	app           *tview.Application
	client        *nats.Client
	config        *config.Config
	pluginManager *plugins.Manager
	readOnly      bool

	// UI components
	pages  *tview.Pages
	header *components.Header
	footer *components.Footer

	// Views
	contextView        *ContextView
	streamListView     *StreamListView
	streamDetailView   *StreamDetailView
	consumerDetailView *ConsumerDetailView
	messageView        *MessageView
	describeView       *DescribeView
	queryBuilderView   *QueryBuilderView
	metricsGraphView   *MetricsGraphView
	streamEditView     *StreamEditView
	consumerEditView   *ConsumerEditView
	helpView           *HelpView

	// State
	currentPage  string
	updateTicker *time.Ticker
}

// NewUIManager creates a new UI manager
func NewUIManager(app *tview.Application, client *nats.Client, cfg *config.Config, pluginMgr *plugins.Manager, readOnly bool) *UIManager {
	ui := &UIManager{
		app:           app,
		client:        client,
		config:        cfg,
		pluginManager: pluginMgr,
		readOnly:      readOnly,
		pages:         tview.NewPages(),
	}

	ui.initComponents()
	ui.setupPages()
	ui.setupKeybindings()

	return ui
}

func (ui *UIManager) initComponents() {
	// Header and footer
	ui.header = components.NewHeader()
	ui.footer = components.NewFooter()
	ui.updateHeader()

	// Initialize views
	ui.contextView = NewContextView(ui)
	ui.streamListView = NewStreamListView(ui)
	ui.streamDetailView = NewStreamDetailView(ui)
	ui.consumerDetailView = NewConsumerDetailView(ui)
	ui.messageView = NewMessageView(ui)
	ui.describeView = NewDescribeView(ui)
	ui.queryBuilderView = NewQueryBuilderView(ui)
	ui.metricsGraphView = NewMetricsGraphView(ui)
	ui.streamEditView = NewStreamEditView(ui)
	ui.consumerEditView = NewConsumerEditView(ui)
	ui.helpView = NewHelpView(ui)
}

func (ui *UIManager) setupPages() {
	// Add all pages
	ui.pages.AddPage("context", ui.contextView.GetPrimitive(), true, true)
	ui.pages.AddPage("streams", ui.streamListView.GetPrimitive(), true, false)
	ui.pages.AddPage("stream-detail", ui.streamDetailView.GetPrimitive(), true, false)
	ui.pages.AddPage("consumer-detail", ui.consumerDetailView.GetPrimitive(), true, false)
	ui.pages.AddPage("messages", ui.messageView.GetPrimitive(), true, false)
	ui.pages.AddPage("describe", ui.describeView.GetPrimitive(), true, false)
	ui.pages.AddPage("query-builder", ui.queryBuilderView.GetPrimitive(), true, false)
	ui.pages.AddPage("metrics-graph", ui.metricsGraphView.GetPrimitive(), true, false)
	ui.pages.AddPage("stream-edit", ui.streamEditView.GetPrimitive(), true, false)
	ui.pages.AddPage("consumer-edit", ui.consumerEditView.GetPrimitive(), true, false)
}

func (ui *UIManager) setupKeybindings() {
	ui.app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// Don't intercept global keys when in query builder (user is typing)
		if ui.currentPage == "query-builder" {
			// Allow Ctrl+C and ? only
			if event.Key() == tcell.KeyCtrlC {
				ui.app.Stop()
				return nil
			}
			if event.Key() == tcell.KeyRune && event.Rune() == '?' {
				ui.ShowHelp()
				return nil
			}
			return event
		}
		
		// Global keybindings for other views
		switch event.Key() {
		case tcell.KeyCtrlC:
			ui.app.Stop()
			return nil
		case tcell.KeyRune:
			switch event.Rune() {
			case '?':
				ui.ShowHelp()
				return nil
			case 'c':
				if ui.currentPage != "context" {
					ui.ShowContextView()
					return nil
				}
			}
		}
		return event
	})
}

// Start starts the UI
func (ui *UIManager) Start() error {
	// Create main layout
	layout := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(ui.header, 1, 0, false).
		AddItem(ui.pages, 0, 1, true).
		AddItem(ui.footer, 1, 0, false)

	// Start auto-refresh ticker
	ui.updateTicker = time.NewTicker(ui.config.GetRefreshInterval())
	go ui.autoRefreshLoop()

	// Set root and run
	ui.app.SetRoot(layout, true).SetFocus(ui.pages)
	return ui.app.Run()
}

func (ui *UIManager) updateHeader() {
	status := ui.header.UpdateStatus(ui.client.IsConnected())
	ui.header.Update(ui.config.CurrentContextName(), status, ui.readOnly)
}

func (ui *UIManager) autoRefreshLoop() {
	for range ui.updateTicker.C {
		ui.app.QueueUpdateDraw(func() {
			ui.updateHeader()
			// Refresh current view (skip messages - manual refresh only)
			switch ui.currentPage {
			case "streams":
				ui.streamListView.Refresh()
			case "stream-detail":
				ui.streamDetailView.Refresh()
			case "consumer-detail":
				ui.consumerDetailView.Refresh()
			case "describe":
				ui.describeView.Refresh()
			// Messages view excluded from auto-refresh (expensive operation)
			}
		})
	}
}

// ShowContextView displays the context selection view
func (ui *UIManager) ShowContextView() {
	ui.currentPage = "context"
	ui.pages.SwitchToPage("context")
	ui.contextView.Refresh()
	ui.app.SetFocus(ui.contextView.GetPrimitive())
}

// ShowStreamList displays the stream list view
func (ui *UIManager) ShowStreamList() {
	ui.currentPage = "streams"
	ui.pages.SwitchToPage("streams")
	ui.streamListView.Refresh()
	ui.app.SetFocus(ui.streamListView.GetPrimitive())
}

// ShowStreamDetail displays the stream detail view
func (ui *UIManager) ShowStreamDetail(streamName string) {
	ui.currentPage = "stream-detail"
	ui.streamDetailView.SetStream(streamName)
	ui.pages.SwitchToPage("stream-detail")
	ui.app.SetFocus(ui.streamDetailView.GetPrimitive())
}

// ShowConsumerDetail displays the consumer detail view
func (ui *UIManager) ShowConsumerDetail(streamName, consumerName string) {
	ui.currentPage = "consumer-detail"
	ui.consumerDetailView.SetConsumer(streamName, consumerName)
	ui.pages.SwitchToPage("consumer-detail")
	ui.app.SetFocus(ui.consumerDetailView.GetPrimitive())
}

// ShowMessages displays the message browser view
func (ui *UIManager) ShowMessages(streamName string) {
	ui.currentPage = "messages"
	ui.messageView.SetStream(streamName)
	ui.pages.SwitchToPage("messages")
	ui.app.SetFocus(ui.messageView.GetPrimitive())
}

// ShowDescribe displays the stream description view
func (ui *UIManager) ShowDescribe(streamName string) {
	ui.currentPage = "describe"
	ui.describeView.SetStream(streamName)
	ui.pages.SwitchToPage("describe")
	ui.app.SetFocus(ui.describeView.GetPrimitive())
}

// ShowQueryBuilder displays the bulk operations query builder
func (ui *UIManager) ShowQueryBuilder() {
	ui.queryBuilderView.Show()
}

// ShowMetricsGraph displays metrics graphs for a stream
func (ui *UIManager) ShowMetricsGraph(streamName string) {
	ui.currentPage = "metrics-graph"
	ui.metricsGraphView.SetStream(streamName)
	ui.pages.SwitchToPage("metrics-graph")
	ui.app.SetFocus(ui.metricsGraphView.GetPrimitive())
}

// ShowStreamEdit displays the stream edit form
func (ui *UIManager) ShowStreamEdit(streamName string) {
	ui.streamEditView.SetStream(streamName)
	ui.streamEditView.Show()
}

// ShowConsumerEdit displays the consumer edit form
func (ui *UIManager) ShowConsumerEdit(streamName, consumerName string) {
	ui.consumerEditView.SetConsumer(streamName, consumerName)
	ui.consumerEditView.Show()
}

// ShowInputDialog displays an input dialog
func (ui *UIManager) ShowInputDialog(title, label, initialValue string, onSubmit func(string)) {
	modal := components.InputModal(title, label, initialValue, onSubmit, func() {
		ui.CloseModal()
	})
	ui.ShowModal(modal)
}

// ShowHelp displays the help modal
func (ui *UIManager) ShowHelp() {
	modal := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(ui.helpView.GetPrimitive(), 30, 1, true).
			AddItem(nil, 0, 1, false), 80, 1, true).
		AddItem(nil, 0, 1, false)

	ui.pages.AddPage("help-modal", modal, true, true)
}

// ShowModal displays a modal dialog
func (ui *UIManager) ShowModal(modal tview.Primitive) {
	ui.pages.AddPage("modal", modal, true, true)
}

// CloseModal closes any open modal
func (ui *UIManager) CloseModal() {
	ui.pages.RemovePage("modal")
	ui.pages.RemovePage("help-modal")
}

// ShowError displays an error message
func (ui *UIManager) ShowError(message string) {
	modal := components.ErrorModal(message, func() {
		ui.CloseModal()
	})
	ui.ShowModal(modal)
}

// SwitchContext switches to a different NATS context
func (ui *UIManager) SwitchContext(contextName string) error {
	// Set the context in config
	if err := ui.config.SetContext(contextName); err != nil {
		return err
	}

	// Close old connection
	ui.client.Close()

	// Create new client with new context
	newClient, err := nats.NewClient(ui.config.CurrentContext())
	if err != nil {
		return fmt.Errorf("failed to connect to new context: %w", err)
	}

	ui.client = newClient
	ui.updateHeader()

	return nil
}

