package ui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// ContextView displays and manages NATS contexts
type ContextView struct {
	ui    *UIManager
	table *tview.Table
}

// NewContextView creates a new context view
func NewContextView(ui *UIManager) *ContextView {
	view := &ContextView{
		ui: ui,
	}

	view.table = tview.NewTable().
		SetBorders(false).
		SetSelectable(true, false).
		SetFixed(1, 0)

	view.table.SetBorder(true).
		SetTitle(" Select Context ").
		SetTitleAlign(tview.AlignCenter)

	view.setupKeybindings()
	view.Refresh()

	return view
}

func (v *ContextView) setupKeybindings() {
	v.table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEnter:
			v.onEnter()
			return nil
		case tcell.KeyRune:
			switch event.Rune() {
			case 'q':
				v.ui.app.Stop()
				return nil
			case 'r':
				v.Refresh()
				return nil
			}
		}
		return event
	})
}

// Refresh updates the context list
func (v *ContextView) Refresh() {
	// Clear table
	v.table.Clear()

	// Set header
	headers := []string{"NAME", "SERVER"}
	for i, header := range headers {
		cell := tview.NewTableCell(header).
			SetTextColor(tcell.ColorYellow).
			SetAlign(tview.AlignLeft).
			SetSelectable(false).
			SetExpansion(1)
		v.table.SetCell(0, i, cell)
	}

	// Add contexts
	currentCtx := v.ui.config.CurrentContextName()
	for i, ctx := range v.ui.config.Contexts {
		row := i + 1
		
		name := ctx.Name
		if ctx.Name == currentCtx {
			name = "> " + name
		} else {
			name = "  " + name
		}

		v.table.SetCell(row, 0, tview.NewTableCell(name).SetExpansion(1))
		v.table.SetCell(row, 1, tview.NewTableCell(ctx.Server).SetExpansion(2))
	}

	v.ui.footer.Update("↑/↓: Navigate  Enter: Select  q: Quit  ?: Help")
}

func (v *ContextView) onEnter() {
	row, _ := v.table.GetSelection()
	if row > 0 && row <= len(v.ui.config.Contexts) {
		ctx := v.ui.config.Contexts[row-1]
		
		// Switch to selected context
		if err := v.ui.SwitchContext(ctx.Name); err != nil {
			v.ui.ShowError(fmt.Sprintf("Failed to switch context: %v", err))
			return
		}
		
		// Navigate to stream list
		v.ui.ShowStreamList()
	}
}

// GetPrimitive returns the primitive for this view
func (v *ContextView) GetPrimitive() tview.Primitive {
	return v.table
}

