package ui

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/shubhamrasal/n2s/internal/models"
	"github.com/shubhamrasal/n2s/internal/ui/components"
)

// StreamListView displays a list of NATS streams
type StreamListView struct {
	ui            *UIManager
	mainFlex      *tview.Flex
	leftFlex      *tview.Flex
	table         *tview.Table
	describePanel *tview.TextView
	searchInput   *tview.InputField
	streams       []*models.Stream
	allStreams    []*models.Stream
	filterText    string
	searching     bool
}

// NewStreamListView creates a new stream list view
func NewStreamListView(ui *UIManager) *StreamListView {
	view := &StreamListView{
		ui:         ui,
		streams:    make([]*models.Stream, 0),
		allStreams: make([]*models.Stream, 0),
	}

	view.table = tview.NewTable().
		SetBorders(false).
		SetSelectable(true, false).
		SetFixed(1, 0).
		SetSelectionChangedFunc(func(row, column int) {
			// Update describe panel when selection changes
			view.updateDescribePanel(row)
		})

	view.table.SetBorder(true).
		SetTitle(" Streams ").
		SetTitleAlign(tview.AlignCenter)

	// Search input field
	view.searchInput = tview.NewInputField().
		SetLabel("Filter: ").
		SetFieldWidth(50).
		SetChangedFunc(func(text string) {
			view.filterText = text
			view.applyFilter()
		})

	view.searchInput.SetBorder(true).
		SetTitle(" Search (ESC to clear) ").
		SetTitleAlign(tview.AlignLeft)

	// Describe panel for right side
	view.describePanel = tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetWordWrap(true)
	view.describePanel.SetBorder(true).
		SetTitle(" Stream Details ").
		SetTitleAlign(tview.AlignCenter)
	view.describePanel.SetText("[gray]Select a stream to view details[white]")

	// Left flex for table and search
	view.leftFlex = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(view.table, 0, 1, true)

	// Main flex - horizontal split (left: list, right: describe)
	view.mainFlex = tview.NewFlex().
		SetDirection(tview.FlexColumn).
		AddItem(view.leftFlex, 0, 2, true).
		AddItem(view.describePanel, 0, 1, false)

	view.setupKeybindings()
	view.setupHeaders()

	return view
}

func (v *StreamListView) setupHeaders() {
	headers := []string{"NAME", "SUBJECTS", "MSGS", "BYTES", "CONSUMERS"}
	for i, header := range headers {
		cell := tview.NewTableCell(header).
			SetTextColor(tcell.ColorYellow).
			SetAlign(tview.AlignLeft).
			SetSelectable(false)
		v.table.SetCell(0, i, cell)
	}
}

func (v *StreamListView) setupKeybindings() {
	v.table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEnter:
			v.onEnter()
			return nil
		case tcell.KeyEsc:
			// If filter is active, clear it first
			if v.filterText != "" {
				v.clearSearch()
				return nil
			}
			// Otherwise go back to context view
			v.ui.ShowContextView()
			return nil
		case tcell.KeyRune:
			switch event.Rune() {
			case '/':
				v.showSearch()
				return nil
			case 'b':
				v.ui.ShowQueryBuilder()
				return nil
			case 'd':
				// Show full-screen describe view
				row, _ := v.table.GetSelection()
				if row > 0 && row <= len(v.streams) {
					stream := v.streams[row-1]
					v.ui.ShowDescribe(stream.Name)
				}
				return nil
			case 'x':
				v.deleteStream()
				return nil
			case 'p':
				v.purgeStream()
				return nil
			case 'r':
				v.Refresh()
				return nil
			case 'm':
				v.viewMessages()
				return nil
			case 'g':
				v.viewMetricsGraph()
				return nil
			case 'e':
				v.editStream()
				return nil
			}
		}
		return event
	})

	v.searchInput.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEsc:
			v.clearSearch()
			return nil
		case tcell.KeyEnter, tcell.KeyTab:
			// Close search but keep filter, and jump to table
			v.closeSearchKeepFilter()
			return nil
		}
		return event
	})
}

// Refresh updates the stream list
func (v *StreamListView) Refresh() {
	// Fetch streams from NATS
	streams, err := v.ui.client.ListStreams()
	if err != nil {
		v.ui.ShowError(fmt.Sprintf("Failed to list streams: %v", err))
		return
	}

	v.allStreams = streams
	v.applyFilter()
}

func (v *StreamListView) showSearch() {
	if v.searching {
		return
	}
	v.searching = true
	v.leftFlex.Clear()
	v.leftFlex.AddItem(v.searchInput, 3, 0, true)
	v.leftFlex.AddItem(v.table, 0, 1, false)
	v.ui.app.SetFocus(v.searchInput)
	v.updateFooter()
}

func (v *StreamListView) clearSearch() {
	v.searching = false
	v.filterText = ""
	v.searchInput.SetText("")
	v.leftFlex.Clear()
	v.leftFlex.AddItem(v.table, 0, 1, true)
	v.ui.app.SetFocus(v.table)
	v.applyFilter()
}

func (v *StreamListView) closeSearchKeepFilter() {
	v.searching = false
	v.leftFlex.Clear()
	v.leftFlex.AddItem(v.table, 0, 1, true)
	v.ui.app.SetFocus(v.table)
	v.updateFooter()
}

func (v *StreamListView) applyFilter() {
	if v.filterText == "" {
		v.streams = v.allStreams
	} else {
		v.streams = make([]*models.Stream, 0)
		filterLower := strings.ToLower(v.filterText)
		for _, stream := range v.allStreams {
			if strings.Contains(strings.ToLower(stream.Name), filterLower) {
				v.streams = append(v.streams, stream)
			}
		}
	}
	v.updateTable()
}

func (v *StreamListView) updateTable() {
	// Clear existing rows (keep header)
	for row := v.table.GetRowCount() - 1; row > 0; row-- {
		v.table.RemoveRow(row)
	}

	// Add stream rows
	for i, stream := range v.streams {
		row := i + 1

		// Format subjects
		subjects := fmt.Sprintf("%v", stream.Subjects)
		if len(subjects) > 30 {
			subjects = subjects[:27] + "..."
		}

		v.table.SetCell(row, 0, tview.NewTableCell(stream.Name))
		v.table.SetCell(row, 1, tview.NewTableCell(subjects))
		v.table.SetCell(row, 2, tview.NewTableCell(formatNumber(stream.Messages)))
		v.table.SetCell(row, 3, tview.NewTableCell(formatBytes(stream.Bytes)))
		v.table.SetCell(row, 4, tview.NewTableCell(fmt.Sprintf("%d", stream.Consumers)))
	}

	v.updateFooter()
}

func (v *StreamListView) updateFooter() {
	if v.searching {
		v.ui.footer.Update("Type to filter  Tab/Enter: Jump to list  ESC: Clear filter")
	} else {
		filterInfo := ""
		if v.filterText != "" {
			filterInfo = fmt.Sprintf(" [Filtered: %d/%d]", len(v.streams), len(v.allStreams))
		}
		v.ui.footer.Update(fmt.Sprintf("Enter: Details  b: Bulk  d: Describe  e: Edit  g: Graphs  m: Messages  x: Delete%s", filterInfo))
	}
}

func (v *StreamListView) onEnter() {
	row, _ := v.table.GetSelection()
	if row > 0 && row <= len(v.streams) {
		stream := v.streams[row-1]
		v.ui.ShowStreamDetail(stream.Name)
	}
}

func (v *StreamListView) viewMessages() {
	row, _ := v.table.GetSelection()
	if row > 0 && row <= len(v.streams) {
		stream := v.streams[row-1]
		v.ui.ShowMessages(stream.Name)
	}
}

func (v *StreamListView) viewMetricsGraph() {
	row, _ := v.table.GetSelection()
	if row > 0 && row <= len(v.streams) {
		stream := v.streams[row-1]
		v.ui.ShowMetricsGraph(stream.Name)
	}
}

func (v *StreamListView) editStream() {
	row, _ := v.table.GetSelection()
	if row > 0 && row <= len(v.streams) {
		stream := v.streams[row-1]
		v.ui.ShowStreamEdit(stream.Name)
	}
}

func (v *StreamListView) deleteStream() {
	if v.ui.readOnly {
		v.ui.ShowError("Cannot delete in read-only mode")
		return
	}

	row, _ := v.table.GetSelection()
	if row > 0 && row <= len(v.streams) {
		stream := v.streams[row-1]

		modal := components.ConfirmModal(
			fmt.Sprintf("Delete stream '%s'?\nThis will delete all messages and consumers.", stream.Name),
			func() {
				v.ui.CloseModal()
				if err := v.ui.client.DeleteStream(stream.Name); err != nil {
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

func (v *StreamListView) purgeStream() {
	if v.ui.readOnly {
		v.ui.ShowError("Cannot purge in read-only mode")
		return
	}

	row, _ := v.table.GetSelection()
	if row > 0 && row <= len(v.streams) {
		stream := v.streams[row-1]

		modal := components.ConfirmModal(
			fmt.Sprintf("Purge all messages from stream '%s'?", stream.Name),
			func() {
				v.ui.CloseModal()
				if err := v.ui.client.PurgeStream(stream.Name); err != nil {
					v.ui.ShowError(fmt.Sprintf("Failed to purge: %v", err))
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

func (v *StreamListView) updateDescribePanel(row int) {
	if row <= 0 || row > len(v.streams) {
		v.describePanel.SetText("[gray]Select a stream to view details[white]")
		return
	}

	stream := v.streams[row-1]

	// Build describe text
	var output strings.Builder

	output.WriteString(fmt.Sprintf("[yellow]%s[white]\n\n", stream.Name))
	output.WriteString(fmt.Sprintf("[cyan]Subjects:[white]\n  %s\n\n", strings.Join(stream.Subjects, "\n  ")))
	output.WriteString(fmt.Sprintf("[cyan]Storage:[white] %s\n", stream.Config.Storage))
	output.WriteString(fmt.Sprintf("[cyan]Retention:[white] %s\n", stream.Config.Retention))
	output.WriteString(fmt.Sprintf("[cyan]Replicas:[white] %d\n\n", stream.Config.Replicas))

	output.WriteString("[yellow]Messages:[white]\n")
	output.WriteString(fmt.Sprintf("  Total: %s\n", formatNumber(stream.State.Messages)))
	output.WriteString(fmt.Sprintf("  Bytes: %s\n", formatBytes(stream.State.Bytes)))
	output.WriteString(fmt.Sprintf("  Range: %d - %d\n\n", stream.State.FirstSeq, stream.State.LastSeq))

	output.WriteString("[yellow]Limits:[white]\n")
	output.WriteString(fmt.Sprintf("  Max Age: %s\n", formatDuration(stream.Config.MaxAge)))
	output.WriteString(fmt.Sprintf("  Max Msgs: %s\n", formatNumber(uint64(stream.Config.MaxMessages))))
	output.WriteString(fmt.Sprintf("  Max Bytes: %s\n\n", formatBytes(uint64(stream.Config.MaxBytes))))

	output.WriteString(fmt.Sprintf("[yellow]Consumers:[white] %d\n", stream.Consumers))

	v.describePanel.SetText(output.String())
	v.describePanel.ScrollToBeginning()
}

// GetPrimitive returns the primitive for this view
func (v *StreamListView) GetPrimitive() tview.Primitive {
	return v.mainFlex
}

// Helper functions
func formatNumber(n uint64) string {
	if n > 1000000 {
		return fmt.Sprintf("%.1fM", float64(n)/1000000)
	} else if n > 1000 {
		return fmt.Sprintf("%.1fK", float64(n)/1000)
	}
	return fmt.Sprintf("%d", n)
}

func formatBytes(b uint64) string {
	if b > 1024*1024*1024 {
		return fmt.Sprintf("%.1fGB", float64(b)/(1024*1024*1024))
	} else if b > 1024*1024 {
		return fmt.Sprintf("%.1fMB", float64(b)/(1024*1024))
	} else if b > 1024 {
		return fmt.Sprintf("%.1fKB", float64(b)/1024)
	}
	return fmt.Sprintf("%dB", b)
}
