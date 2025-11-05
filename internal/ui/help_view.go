package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// HelpView displays keybinding help
type HelpView struct {
	ui       *UIManager
	textView *tview.TextView
}

// NewHelpView creates a new help view
func NewHelpView(ui *UIManager) *HelpView {
	view := &HelpView{
		ui: ui,
	}

	view.textView = tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetText(view.getHelpText())

	view.textView.SetBorder(true).
		SetTitle(" N2S Keybindings - Press Esc to close ").
		SetTitleAlign(tview.AlignCenter)

	view.setupKeybindings()

	return view
}

func (v *HelpView) getHelpText() string {
	return `
[yellow]Global Keybindings[white]
  Ctrl+C     Quit application
  ?          Show this help
  c          Switch to context selection
  Esc        Go back / Cancel

[yellow]Context Selection View[white]
  ↑/↓, j/k   Navigate contexts
  Enter      Connect to selected context
  q          Quit

[yellow]Stream List View[white]
  ↑/↓, j/k   Navigate streams
  Enter      View stream details
  /          Filter streams
  d          Describe Stream
  x          Delete stream (with confirmation)
  p          Purge stream messages (with confirmation)
  m          View messages
  r          Refresh
  Esc        Back to context selection

[yellow]Stream Detail View[white]
  ↑/↓, j/k   Navigate consumers
  Enter      View consumer details
  d          Describe Stream
  m          View messages in stream
  x          Delete selected consumer
  Esc        Back to stream list

[yellow]Describe View[white]
  r          Refresh
  Esc        Back to stream detail

[yellow]Metrics Graphs (g)[white]
  r          Refresh
  Esc        Back to stream list

[yellow]Consumer Detail View[white]
  d          Delete consumer (with confirmation)
  Esc        Back to stream detail

[yellow]Message Browser View[white]
  ↑/↓, j/k   Navigate messages
  Enter      View message detail
  Esc        Back

[yellow]Tips[white]
  • Use --read-only flag to prevent deletions in production
  • Press 'r' to manually refresh any view
  • Views auto-refresh every 2 seconds
  • Use 'c' from anywhere to switch contexts
`
}

func (v *HelpView) setupKeybindings() {
	v.textView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEsc:
			v.ui.CloseModal()
			return nil
		case tcell.KeyRune:
			if event.Rune() == 'q' {
				v.ui.CloseModal()
				return nil
			}
		}
		return event
	})
}

// GetPrimitive returns the primitive for this view
func (v *HelpView) GetPrimitive() tview.Primitive {
	return v.textView
}

