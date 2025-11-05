package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/shubhamrasal/n2s/internal/models"
	"github.com/shubhamrasal/n2s/internal/ui/components"
	"gopkg.in/yaml.v3"
)

// QueryBuilderView provides bulk operations with filtering
type QueryBuilderView struct {
	ui           *UIManager
	mainFlex     *tview.Flex
	form         *tview.Form
	previewTable *tview.Table
	statusText   *tview.TextView

	// Filter criteria
	namePattern   string
	ageOp         string
	ageValue      string
	ageUnit       string
	consumerOp    string
	consumerValue string
	messagesOp    string
	messagesValue string

	// Preview data
	matchedStreams []*models.Stream
	sortColumn     int
	sortAscending  bool
}

// NewQueryBuilderView creates a new query builder
func NewQueryBuilderView(ui *UIManager) *QueryBuilderView {
	view := &QueryBuilderView{
		ui:            ui,
		namePattern:   "*",
		ageOp:         "any",
		ageValue:      "",
		ageUnit:       "h",
		consumerOp:    "any",
		consumerValue: "",
		messagesOp:    "any",
		messagesValue: "",
		sortColumn:    0,
		sortAscending: true,
	}

	view.buildUI()
	view.setupKeybindings()

	return view
}

func (v *QueryBuilderView) buildUI() {
	// Create form for filter criteria
	v.form = tview.NewForm()

	// Name pattern input
	v.form.AddInputField("Name Pattern", v.namePattern, 30, nil, func(text string) {
		v.namePattern = text
	})

	// Age operator dropdown
	ageOps := []string{"any", ">", "<", "="}
	v.form.AddDropDown("Age Operator", ageOps, 0, func(option string, optionIndex int) {
		v.ageOp = option
	})

	// Age value input
	v.form.AddInputField("Age Value", v.ageValue, 10, tview.InputFieldInteger, func(text string) {
		v.ageValue = text
	})

	// Age unit dropdown
	ageUnits := []string{"m", "h"}
	v.form.AddDropDown("Age Unit", ageUnits, 1, func(option string, optionIndex int) {
		v.ageUnit = option
	})

	// Consumer operator dropdown
	consumerOps := []string{"any", "=", ">", "<"}
	v.form.AddDropDown("Consumer Op", consumerOps, 0, func(option string, optionIndex int) {
		v.consumerOp = option
	})

	// Consumer value input
	v.form.AddInputField("Consumer Value", v.consumerValue, 10, tview.InputFieldInteger, func(text string) {
		v.consumerValue = text
	})

	// Messages operator dropdown
	messagesOps := []string{"any", "=", ">", "<"}
	v.form.AddDropDown("Messages Op", messagesOps, 0, func(option string, optionIndex int) {
		v.messagesOp = option
	})

	// Messages value input
	v.form.AddInputField("Messages Value", v.messagesValue, 15, tview.InputFieldInteger, func(text string) {
		v.messagesValue = text
	})

	// Action buttons
	v.form.AddButton("[ Preview Matches ]", func() {
		v.previewMatches()
		v.ui.app.SetFocus(v.form)
	})

	v.form.AddButton("[ Delete All ]", func() {
		v.deleteMatched()
	})

	v.form.AddButton("[ Purge All ]", func() {
		v.purgeMatched()
	})

	v.form.AddButton("[ Load Filter ]", func() {
		v.showLoadFilterDialog()
	})

	v.form.AddButton("[ Save Filter ]", func() {
		v.saveFilter()
	})

	v.form.AddButton("[ Clear Filter ]", func() {
		v.clearFilter()
	})

	v.form.AddButton("[ Cancel ]", func() {
		v.ui.ShowStreamList()
	})

	v.form.SetBorder(true).
		SetTitle(" Bulk Operation - Filter Criteria ").
		SetTitleAlign(tview.AlignCenter)

	// Preview table
	v.previewTable = tview.NewTable().
		SetBorders(false).
		SetSelectable(true, false).
		SetFixed(1, 0)

	v.previewTable.SetBorder(true).
		SetTitle(" Preview - Click column header to sort ").
		SetTitleAlign(tview.AlignCenter)

	v.setupPreviewHeaders()

	// Status text
	v.statusText = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)
	v.statusText.SetBorder(true)
	v.updateStatus(0)

	// Layout: form on left, preview on right
	leftFlex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(v.form, 0, 1, true).
		AddItem(v.statusText, 3, 0, false)

	v.mainFlex = tview.NewFlex().
		SetDirection(tview.FlexColumn).
		AddItem(leftFlex, 0, 1, true).
		AddItem(v.previewTable, 0, 2, false)
}

func (v *QueryBuilderView) setupPreviewHeaders() {
	headers := []string{"NAME", "AGE", "MSGS", "CONSUMERS"}
	for i, header := range headers {
		cell := tview.NewTableCell(header).
			SetTextColor(tcell.ColorYellow).
			SetAlign(tview.AlignLeft).
			SetSelectable(true)
		v.previewTable.SetCell(0, i, cell)
	}
}

func (v *QueryBuilderView) setupKeybindings() {
	v.mainFlex.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			v.ui.ShowStreamList()
			return nil
		}
		return event
	})

	// Make header clickable for sorting
	v.previewTable.SetSelectedFunc(func(row, column int) {
		if row == 0 {
			// Header clicked - sort by this column
			if v.sortColumn == column {
				v.sortAscending = !v.sortAscending
			} else {
				v.sortColumn = column
				v.sortAscending = true
			}
			v.sortPreview()
		}
	})
}

func (v *QueryBuilderView) previewMatches() {
	// Get all streams
	streams, err := v.ui.client.ListStreams()
	if err != nil {
		v.ui.ShowError(fmt.Sprintf("Failed to list streams: %v", err))
		return
	}

	// Filter streams
	v.matchedStreams = v.filterStreams(streams)

	// Update preview table
	v.updatePreviewTable()
	v.updateStatus(len(v.matchedStreams))
}

func (v *QueryBuilderView) filterStreams(streams []*models.Stream) []*models.Stream {
	var matched []*models.Stream

	for _, stream := range streams {
		if v.matchesFilter(stream) {
			matched = append(matched, stream)
		}
	}

	return matched
}

func (v *QueryBuilderView) matchesFilter(stream *models.Stream) bool {
	// Name pattern matching (only if not wildcard)
	if v.namePattern != "*" && v.namePattern != "" {
		// Convert wildcard to regex
		pattern := strings.ReplaceAll(v.namePattern, "*", ".*")
		pattern = "^" + pattern + "$"
		matched, err := regexp.MatchString(pattern, stream.Name)
		if err != nil || !matched {
			return false
		}
	}

	// Age filtering (only if operator set and value provided)
	if v.ageOp != "any" && v.ageValue != "" {
		age := time.Since(stream.State.FirstTime)
		var targetDuration time.Duration

		// Parse age value
		var ageVal int
		if n, _ := fmt.Sscanf(v.ageValue, "%d", &ageVal); n == 0 {
			return true // Skip if invalid value
		}

		if v.ageUnit == "m" {
			targetDuration = time.Duration(ageVal) * time.Minute
		} else {
			targetDuration = time.Duration(ageVal) * time.Hour
		}

		if !v.compareAge(age, v.ageOp, targetDuration) {
			return false
		}
	}

	// Consumer filtering (only if operator set and value provided)
	if v.consumerOp != "any" && v.consumerValue != "" {
		var consVal int
		if n, _ := fmt.Sscanf(v.consumerValue, "%d", &consVal); n == 0 {
			return true // Skip if invalid value
		}

		if !v.compareInt(stream.Consumers, v.consumerOp, consVal) {
			return false
		}
	}

	// Messages filtering (only if operator set and value provided)
	if v.messagesOp != "any" && v.messagesValue != "" {
		var msgVal int64
		if n, _ := fmt.Sscanf(v.messagesValue, "%d", &msgVal); n == 0 {
			return true // Skip if invalid value
		}

		if !v.compareInt64(int64(stream.State.Messages), v.messagesOp, msgVal) {
			return false
		}
	}

	return true
}

func (v *QueryBuilderView) compareAge(actual time.Duration, op string, target time.Duration) bool {
	switch op {
	case ">":
		return actual > target
	case "<":
		return actual < target
	case "=":
		// Within 10% tolerance
		return actual >= target*9/10 && actual <= target*11/10
	}
	return true
}

func (v *QueryBuilderView) compareInt(actual int, op string, target int) bool {
	switch op {
	case "=":
		return actual == target
	case ">":
		return actual > target
	case "<":
		return actual < target
	}
	return true
}

func (v *QueryBuilderView) compareInt64(actual int64, op string, target int64) bool {
	switch op {
	case "=":
		return actual == target
	case ">":
		return actual > target
	case "<":
		return actual < target
	}
	return true
}

func (v *QueryBuilderView) updatePreviewTable() {
	// Clear existing rows
	for row := v.previewTable.GetRowCount() - 1; row > 0; row-- {
		v.previewTable.RemoveRow(row)
	}

	// Add matched streams
	for i, stream := range v.matchedStreams {
		row := i + 1
		age := time.Since(stream.State.FirstTime)

		v.previewTable.SetCell(row, 0, tview.NewTableCell(stream.Name))
		v.previewTable.SetCell(row, 1, tview.NewTableCell(formatDuration(age)))
		v.previewTable.SetCell(row, 2, tview.NewTableCell(formatNumber(stream.State.Messages)))
		v.previewTable.SetCell(row, 3, tview.NewTableCell(fmt.Sprintf("%d", stream.Consumers)))
	}
}

func (v *QueryBuilderView) sortPreview() {
	if len(v.matchedStreams) == 0 {
		return
	}

	sort.Slice(v.matchedStreams, func(i, j int) bool {
		var less bool

		switch v.sortColumn {
		case 0: // Name
			less = v.matchedStreams[i].Name < v.matchedStreams[j].Name
		case 1: // Age
			less = v.matchedStreams[i].State.FirstTime.Before(v.matchedStreams[j].State.FirstTime)
		case 2: // Messages
			less = v.matchedStreams[i].State.Messages < v.matchedStreams[j].State.Messages
		case 3: // Consumers
			less = v.matchedStreams[i].Consumers < v.matchedStreams[j].Consumers
		}

		if !v.sortAscending {
			less = !less
		}

		return less
	})

	v.updatePreviewTable()
}

func (v *QueryBuilderView) updateStatus(count int) {
	if count == 0 {
		v.statusText.SetText("[gray]Press 'Preview Matches' to see results[white]")
	} else {
		v.statusText.SetText(fmt.Sprintf("[yellow]%d streams match criteria[white]", count))
	}
}

func (v *QueryBuilderView) deleteMatched() {
	if len(v.matchedStreams) == 0 {
		v.ui.ShowError("No streams matched. Press 'Preview Matches' first.")
		return
	}

	if v.ui.readOnly {
		v.ui.ShowError("Cannot delete in read-only mode")
		return
	}

	// Create confirmation with stream names
	streamNames := []string{}
	for i, s := range v.matchedStreams {
		if i < 5 {
			streamNames = append(streamNames, "  - "+s.Name)
		}
	}
	if len(v.matchedStreams) > 5 {
		streamNames = append(streamNames, fmt.Sprintf("  ... and %d more", len(v.matchedStreams)-5))
	}

	message := fmt.Sprintf("Delete %d streams?\n\n%s\n\nThis action cannot be undone!",
		len(v.matchedStreams),
		strings.Join(streamNames, "\n"))

	modal := components.ConfirmModal(
		message,
		func() {
			v.ui.CloseModal()
			v.performBulkDelete()
		},
		func() {
			v.ui.CloseModal()
		},
	)

	v.ui.ShowModal(modal)
}

func (v *QueryBuilderView) performBulkDelete() {
	successCount := 0
	failCount := 0

	for _, stream := range v.matchedStreams {
		if err := v.ui.client.DeleteStream(stream.Name); err != nil {
			failCount++
		} else {
			successCount++
		}
	}

	// Show success message (not error)
	message := fmt.Sprintf("Bulk Delete Complete\n\nDeleted: %d streams\nFailed: %d streams",
		successCount, failCount)

	modal := components.InfoModal("Bulk Delete Result", message, func() {
		v.ui.CloseModal()
		v.ui.ShowStreamList()
	})
	v.ui.ShowModal(modal)
}

func (v *QueryBuilderView) purgeMatched() {
	if len(v.matchedStreams) == 0 {
		v.ui.ShowError("No streams matched. Press 'Preview Matches' first.")
		return
	}

	if v.ui.readOnly {
		v.ui.ShowError("Cannot purge in read-only mode")
		return
	}

	// Create confirmation
	streamNames := []string{}
	for i, s := range v.matchedStreams {
		if i < 5 {
			streamNames = append(streamNames, "  - "+s.Name)
		}
	}
	if len(v.matchedStreams) > 5 {
		streamNames = append(streamNames, fmt.Sprintf("  ... and %d more", len(v.matchedStreams)-5))
	}

	message := fmt.Sprintf("Purge all messages from %d streams?\n\n%s\n\nConsumers will remain, only messages deleted.",
		len(v.matchedStreams),
		strings.Join(streamNames, "\n"))

	modal := components.ConfirmModal(
		message,
		func() {
			v.ui.CloseModal()
			v.performBulkPurge()
		},
		func() {
			v.ui.CloseModal()
		},
	)

	v.ui.ShowModal(modal)
}

func (v *QueryBuilderView) performBulkPurge() {
	successCount := 0
	failCount := 0

	for _, stream := range v.matchedStreams {
		if err := v.ui.client.PurgeStream(stream.Name); err != nil {
			failCount++
		} else {
			successCount++
		}
	}

	// Show result
	message := fmt.Sprintf("Bulk Purge Complete\n\nPurged: %d streams\nFailed: %d streams",
		successCount, failCount)

	modal := components.InfoModal("Bulk Purge Result", message, func() {
		v.ui.CloseModal()
		// Restore focus after closing modal
		v.ui.app.SetFocus(v.form)
	})
	v.ui.ShowModal(modal)

	// Refresh preview to show updated message counts
	v.previewMatches()
}

func (v *QueryBuilderView) saveFilter() {
	// Show input dialog for filter name
	v.ui.ShowInputDialog("Save Filter", "Filter Name:", "", func(name string) {
		if name == "" {
			v.ui.ShowError("Filter name cannot be empty")
			return
		}

		filter := models.SavedFilter{
			Name:          name,
			NamePattern:   v.namePattern,
			AgeOp:         v.ageOp,
			AgeValue:      0,
			AgeUnit:       v.ageUnit,
			ConsumerOp:    v.consumerOp,
			ConsumerValue: 0,
			MessagesOp:    v.messagesOp,
			MessagesValue: 0,
		}

		// Parse integer values
		fmt.Sscanf(v.ageValue, "%d", &filter.AgeValue)
		fmt.Sscanf(v.consumerValue, "%d", &filter.ConsumerValue)
		fmt.Sscanf(v.messagesValue, "%d", &filter.MessagesValue)

		// Save to config file
		if err := v.saveFilterToFile(filter); err != nil {
			v.ui.ShowError(fmt.Sprintf("Failed to save filter: %v", err))
			return
		}

		// Show success message
		modal := components.InfoModal("Filter Saved",
			fmt.Sprintf("Filter '%s' saved successfully!\n\nSaved to: ~/.config/n2s/filters.yaml", name),
			func() {
				v.ui.CloseModal()
			})
		v.ui.ShowModal(modal)
	})
}

func (v *QueryBuilderView) saveFilterToFile(filter models.SavedFilter) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	filterPath := filepath.Join(homeDir, ".config", "n2s", "filters.yaml")

	// Ensure directory exists first
	if err := os.MkdirAll(filepath.Dir(filterPath), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Load existing filters
	var config models.FilterConfig
	if data, err := os.ReadFile(filterPath); err == nil {
		yaml.Unmarshal(data, &config)
	}

	// Check for duplicate filter name
	for _, f := range config.Filters {
		if f.Name == filter.Name {
			return fmt.Errorf("filter '%s' already exists", filter.Name)
		}
	}

	// Add new filter
	config.Filters = append(config.Filters, filter)

	// Save
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal filters: %w", err)
	}

	if err := os.WriteFile(filterPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write filters file: %w", err)
	}

	return nil
}

func (v *QueryBuilderView) loadSavedFilters() ([]models.SavedFilter, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	filterPath := filepath.Join(homeDir, ".config", "n2s", "filters.yaml")

	data, err := os.ReadFile(filterPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []models.SavedFilter{}, nil
		}
		return nil, err
	}

	var config models.FilterConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return config.Filters, nil
}

func (v *QueryBuilderView) showLoadFilterDialog() {
	// Load saved filters
	savedFilters, err := v.loadSavedFilters()
	if err != nil {
		v.ui.ShowError(fmt.Sprintf("Failed to load filters: %v", err))
		return
	}

	if len(savedFilters) == 0 {
		v.ui.ShowError("No saved filters found.\n\nSave a filter first using 'Save Filter' button.")
		return
	}

	// Create a list of filter names
	filterNames := make([]string, len(savedFilters))
	for i, f := range savedFilters {
		filterNames[i] = f.Name
	}

	// Create form for selecting filter
	selectForm := tview.NewForm()
	var selectedFilterIndex int
	selectForm.AddDropDown("Saved Filters", filterNames, 0, func(option string, optionIndex int) {
		selectedFilterIndex = optionIndex
	})
	selectForm.AddButton("[ Load ]", func() {
		// Load the selected filter
		if selectedFilterIndex >= 0 && selectedFilterIndex < len(savedFilters) {
			v.ui.CloseModal()
			v.loadFilter(savedFilters[selectedFilterIndex])
		}
	})
	selectForm.AddButton("[ Cancel ]", func() {
		v.ui.CloseModal()
	})

	selectForm.SetBorder(true).
		SetTitle(" Load Saved Filter ").
		SetTitleAlign(tview.AlignCenter)

	// Center the form
	centered := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(selectForm, 8, 1, true).
			AddItem(nil, 0, 1, false), 50, 1, true).
		AddItem(nil, 0, 1, false)

	v.ui.ShowModal(centered)
}

func (v *QueryBuilderView) loadFilter(filter models.SavedFilter) {
	v.namePattern = filter.NamePattern
	v.ageOp = filter.AgeOp
	if filter.AgeValue > 0 {
		v.ageValue = fmt.Sprintf("%d", filter.AgeValue)
	} else {
		v.ageValue = ""
	}
	v.ageUnit = filter.AgeUnit
	v.consumerOp = filter.ConsumerOp
	if filter.ConsumerValue > 0 || filter.ConsumerOp == "=" {
		v.consumerValue = fmt.Sprintf("%d", filter.ConsumerValue)
	} else {
		v.consumerValue = ""
	}
	v.messagesOp = filter.MessagesOp
	if filter.MessagesValue > 0 || filter.MessagesOp == "=" {
		v.messagesValue = fmt.Sprintf("%d", filter.MessagesValue)
	} else {
		v.messagesValue = ""
	}

	// Rebuild form with loaded values
	v.form.Clear(true)
	v.buildFormFields()

	// Auto-preview
	v.previewMatches()
}

// Show shows the query builder
func (v *QueryBuilderView) Show() {
	v.ui.currentPage = "query-builder"
	v.ui.pages.SwitchToPage("query-builder")
	v.ui.app.SetFocus(v.form)
	v.ui.footer.Update("Tab: Navigate fields  Enter: Activate/Open dropdown  Arrows: In dropdowns  Esc: Cancel")
}

func (v *QueryBuilderView) clearFilter() {
	// Reset all filter values to defaults
	v.namePattern = "*"
	v.ageOp = "any"
	v.ageValue = ""
	v.ageUnit = "h"
	v.consumerOp = "any"
	v.consumerValue = ""
	v.messagesOp = "any"
	v.messagesValue = ""

	// Clear matched streams
	v.matchedStreams = nil

	// Rebuild form with cleared values
	v.form.Clear(true)
	v.buildFormFields()

	// Clear preview
	for row := v.previewTable.GetRowCount() - 1; row > 0; row-- {
		v.previewTable.RemoveRow(row)
	}

	v.updateStatus(0)

	// IMPORTANT: Restore focus to form
	v.ui.app.SetFocus(v.form)
	v.ui.app.ForceDraw()
}

func (v *QueryBuilderView) buildFormFields() {
	// Name pattern input
	v.form.AddInputField("Name Pattern", v.namePattern, 30, nil, func(text string) {
		v.namePattern = text
	})

	// Age operator dropdown
	ageOps := []string{"any", ">", "<", "="}
	v.form.AddDropDown("Age Operator", ageOps, 0, func(option string, optionIndex int) {
		v.ageOp = option
	})

	// Age value input
	v.form.AddInputField("Age Value", v.ageValue, 10, tview.InputFieldInteger, func(text string) {
		v.ageValue = text
	})

	// Age unit dropdown
	ageUnits := []string{"m", "h"}
	v.form.AddDropDown("Age Unit", ageUnits, 1, func(option string, optionIndex int) {
		v.ageUnit = option
	})

	// Consumer operator dropdown
	consumerOps := []string{"any", "=", ">", "<"}
	v.form.AddDropDown("Consumer Op", consumerOps, 0, func(option string, optionIndex int) {
		v.consumerOp = option
	})

	// Consumer value input
	v.form.AddInputField("Consumer Value", v.consumerValue, 10, tview.InputFieldInteger, func(text string) {
		v.consumerValue = text
	})

	// Messages operator dropdown
	messagesOps := []string{"any", "=", ">", "<"}
	v.form.AddDropDown("Messages Op", messagesOps, 0, func(option string, optionIndex int) {
		v.messagesOp = option
	})

	// Messages value input
	v.form.AddInputField("Messages Value", v.messagesValue, 15, tview.InputFieldInteger, func(text string) {
		v.messagesValue = text
	})

	// Action buttons
	v.form.AddButton("[ Preview Matches ]", func() {
		v.previewMatches()
		v.ui.app.SetFocus(v.form)
	})

	v.form.AddButton("[ Delete All ]", func() {
		v.deleteMatched()
	})

	v.form.AddButton("[ Purge All ]", func() {
		v.purgeMatched()
	})

	v.form.AddButton("[ Load Filter ]", func() {
		v.showLoadFilterDialog()
	})

	v.form.AddButton("[ Save Filter ]", func() {
		v.saveFilter()
	})

	v.form.AddButton("[ Clear Filter ]", func() {
		v.clearFilter()
	})

	v.form.AddButton("[ Cancel ]", func() {
		v.ui.ShowStreamList()
	})
}

// GetPrimitive returns the primitive for this view
func (v *QueryBuilderView) GetPrimitive() tview.Primitive {
	return v.mainFlex
}
