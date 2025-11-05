package ui

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/guptarohit/asciigraph"
	"github.com/rivo/tview"
	"github.com/shubhamrasal/n2s/internal/models"
)

// MetricsGraphView displays time-series metrics graphs
type MetricsGraphView struct {
	ui           *UIManager
	mainFlex     *tview.Flex
	headerView   *tview.TextView
	graphPanels  map[string]*tview.TextView // 6 graph panels
	streamName   string
	consumerName string
	timeRange    string
	metricsData  *models.MetricsData
	loading      bool
}

// NewMetricsGraphView creates a new metrics graph view
func NewMetricsGraphView(ui *UIManager) *MetricsGraphView {
	view := &MetricsGraphView{
		ui:          ui,
		timeRange:   "1h", // Default
		graphPanels: make(map[string]*tview.TextView),
	}

	// Header with stream info
	view.headerView = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft)

	// Create 6 graph panels
	panelNames := []string{
		"stream_bytes", "stream_messages", "message_rate",
		"consumer_delivered", "consumer_pending", "consumer_ack_pending",
	}

	for _, name := range panelNames {
		panel := tview.NewTextView().
			SetDynamicColors(true).
			SetScrollable(false).
			SetWordWrap(false)
		panel.SetBorder(true)
		view.graphPanels[name] = panel
	}

	// Create grid layout: 2 rows x 3 columns
	row1 := tview.NewFlex().
		AddItem(view.graphPanels["stream_bytes"], 0, 1, false).
		AddItem(view.graphPanels["stream_messages"], 0, 1, false).
		AddItem(view.graphPanels["message_rate"], 0, 1, false)

	row2 := tview.NewFlex().
		AddItem(view.graphPanels["consumer_delivered"], 0, 1, false).
		AddItem(view.graphPanels["consumer_pending"], 0, 1, false).
		AddItem(view.graphPanels["consumer_ack_pending"], 0, 1, false)

	view.mainFlex = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(view.headerView, 3, 0, false).
		AddItem(row1, 0, 1, false).
		AddItem(row2, 0, 1, false)

	view.setupKeybindings()

	return view
}

func (v *MetricsGraphView) setupKeybindings() {
	v.mainFlex.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEsc:
			v.ui.ShowStreamList()
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

// SetStream sets the stream to show metrics for
func (v *MetricsGraphView) SetStream(streamName string) {
	v.streamName = streamName
	v.consumerName = "" // Show all consumers for stream
	v.Refresh()
}

// SetConsumer sets specific consumer to show metrics for
func (v *MetricsGraphView) SetConsumer(streamName, consumerName string) {
	v.streamName = streamName
	v.consumerName = consumerName
	v.Refresh()
}

// Refresh fetches and displays metrics
func (v *MetricsGraphView) Refresh() {
	if v.streamName == "" {
		return
	}

	// Prevent concurrent refreshes
	if v.loading {
		return
	}
	v.loading = true

	// Show loading in all panels
	for _, panel := range v.graphPanels {
		panel.SetText("\n[yellow]Loading...[white]")
	}

	// Fetch metrics in background
	go func() {
		var metricsData *models.MetricsData
		var err error

		// Check if plugin is configured for current context
		pluginName := v.ui.config.CurrentContext().MetricsPlugin
		if pluginName == "" {
			v.ui.app.QueueUpdateDraw(func() {
				v.loading = false
				v.showNoPluginMessage()
			})
			return
		}

		// Get plugin
		plugin, pluginErr := v.ui.pluginManager.GetPlugin(pluginName)
		if pluginErr != nil {
			v.ui.app.QueueUpdateDraw(func() {
				v.loading = false
				v.showPluginError(pluginErr)
			})
			return
		}

		// Fetch metrics
		if v.consumerName != "" {
			metricsData, err = plugin.GetConsumerMetrics(v.streamName, v.consumerName, v.timeRange)
		} else {
			metricsData, err = plugin.GetStreamMetrics(v.streamName, v.timeRange)
		}

		// Update UI
		v.ui.app.QueueUpdateDraw(func() {
			v.loading = false

			if err != nil {
				v.showError(err)
				return
			}

			v.metricsData = metricsData
			v.renderGraphs()
		})
	}()
}

func (v *MetricsGraphView) renderGraphs() {
	if v.metricsData == nil {
		v.showNoData()
		return
	}

	// Update header
	header := fmt.Sprintf("[yellow]Stream:[white] %s  [yellow]Time Range:[white] %s  [yellow]Updated:[white] %s",
		v.metricsData.StreamName, v.timeRange, v.metricsData.FetchTime.Format("15:04:05"))
	v.headerView.SetText(header)

	// Define panel configs: Row 1 (Stream metrics)
	panelConfigs := map[string]struct {
		title string
		key   string
	}{
		"stream_bytes":         {title: "Stream Size (bytes)", key: "stream_bytes"},
		"stream_messages":      {title: "Stream Message Count", key: "stream_messages"},
		"message_rate":         {title: "Message Rate (msg/s)", key: "message_rate"},
		"consumer_delivered":   {title: "Consumer Delivered", key: "consumer_delivered"},
		"consumer_pending":     {title: "Consumer Pending", key: "consumer_pending"},
		"consumer_ack_pending": {title: "Acks Pending", key: "consumer_ack_pending"},
	}

	// Render each panel
	for panelName, config := range panelConfigs {
		panel := v.graphPanels[panelName]
		if series, ok := v.metricsData.Metrics[config.key]; ok && len(series) > 0 {
			v.renderPanelGraph(panel, config.title, series)
		} else {
			panel.SetTitle(fmt.Sprintf(" %s ", config.title))
			panel.SetText("\n[gray]No data[white]")
		}
	}

	v.ui.footer.Update("r: Refresh  Esc/q: Back  [Auto-refresh: 1m]")
}

func (v *MetricsGraphView) showNoData() {
	for _, panel := range v.graphPanels {
		panel.SetText("\n[gray]No data[white]")
	}
}

func (v *MetricsGraphView) renderPanelGraph(panel *tview.TextView, title string, series []models.MetricSeries) {
	panel.SetTitle(fmt.Sprintf(" %s ", title))

	if len(series) == 0 {
		panel.SetText("\n[gray]No data[white]")
		return
	}

	var output strings.Builder

	for _, s := range series {
		if len(s.Points) == 0 {
			continue
		}

		// Calculate stats
		current := s.Points[len(s.Points)-1]
		max := current
		min := current
		sum := 0.0
		for _, p := range s.Points {
			if p > max {
				max = p
			}
			if p < min {
				min = p
			}
			sum += p
		}
		avg := sum / float64(len(s.Points))

		// Get terminal size and calculate graph dimensions
		_, _, width, height := panel.GetInnerRect()

		// Calculate graph size based on available space
		// Width: account for Y-axis labels (~12 chars) and margins
		graphWidth := width - 15
		if graphWidth < 30 {
			graphWidth = 30
		}
		if graphWidth > 100 {
			graphWidth = 100
		}

		// Height: leave room for title, consumer name, caption
		graphHeight := height - 5
		if graphHeight < 8 {
			graphHeight = 8
		}
		if graphHeight > 20 {
			graphHeight = 20
		}

		// Render responsive graph
		graph := asciigraph.Plot(s.Points,
			asciigraph.Height(graphHeight),
			asciigraph.Width(graphWidth),
			asciigraph.Caption(fmt.Sprintf("%s | ↑%s ↓%s ~%s",
				formatMetricValue(current),
				formatMetricValue(max),
				formatMetricValue(min),
				formatMetricValue(avg))))

		// Show consumer name if multiple consumers
		if len(series) > 1 || s.Name != v.streamName {
			output.WriteString(fmt.Sprintf("[cyan]%s[white]\n", s.Name))
		}
		output.WriteString(graph)
		output.WriteString("\n")
	}

	panel.SetText(output.String())
}

func (v *MetricsGraphView) showNoPluginMessage() {
	v.headerView.SetText("[yellow]No Metrics Plugin Configured[white]")

	message := `

  To enable metrics graphs:

  1. Create ~/.config/n2s/plugins.yaml
  2. Add Prometheus credentials
  3. Link in context config
  
  See bin/plugins.yaml.example for template
`
	for _, panel := range v.graphPanels {
		panel.SetTitle(" Info ")
		panel.SetText(message)
	}
}

func (v *MetricsGraphView) showPluginError(err error) {
	v.headerView.SetText(fmt.Sprintf("[red]Plugin Error[white]"))
	message := fmt.Sprintf("\n%v", err)
	for _, panel := range v.graphPanels {
		panel.SetTitle(" Error ")
		panel.SetText(message)
	}
}

func (v *MetricsGraphView) showError(err error) {
	v.headerView.SetText("[red]Failed to fetch metrics[white]")
	message := fmt.Sprintf("\n%v", err)
	for _, panel := range v.graphPanels {
		panel.SetTitle(" Error ")
		panel.SetText(message)
	}
}

// GetPrimitive returns the primitive for this view
func (v *MetricsGraphView) GetPrimitive() tview.Primitive {
	return v.mainFlex
}

// formatMetricValue formats large numbers to human-readable format
func formatMetricValue(val float64) string {
	if val >= 1000000000 {
		return fmt.Sprintf("%.2fB", val/1000000000)
	} else if val >= 1000000 {
		return fmt.Sprintf("%.2fM", val/1000000)
	} else if val >= 1000 {
		return fmt.Sprintf("%.2fK", val/1000)
	} else if val >= 1 {
		return fmt.Sprintf("%.1f", val)
	} else {
		return fmt.Sprintf("%.3f", val)
	}
}
