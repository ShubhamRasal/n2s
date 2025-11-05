package components

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// ConfirmModal creates a confirmation dialog
func ConfirmModal(message string, onConfirm, onCancel func()) *tview.Modal {
	// WORKAROUND: tview modal buttons have reversed behavior
	// We need to swap the callbacks to fix it
	// Visual: [ Yes ] [ No ]
	// Expected: Yes=confirm, No=cancel
	// But tview does: Yes=index 1, No=index 0 (somehow reversed)
	modal := tview.NewModal().
		SetText(message).
		AddButtons([]string{"[ Yes ]", "[ No ]"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {

			if buttonLabel == "[ Yes ]" && onConfirm != nil {
				onConfirm()
			} else if buttonLabel == "[ No ]" && onCancel != nil {
				onCancel()
			}
		})

	// set focus to the second button
	modal.SetFocus(1)

	return modal
}

// ErrorModal creates an error message dialog
func ErrorModal(message string, onDismiss func()) *tview.Modal {
	modal := tview.NewModal().
		SetText("Error: " + message).
		AddButtons([]string{"OK"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			if onDismiss != nil {
				onDismiss()
			}
		})

	modal.SetBackgroundColor(tcell.ColorDefault)
	modal.SetButtonBackgroundColor(tcell.ColorRed)
	modal.SetButtonTextColor(tcell.ColorWhite)
	return modal
}

// InfoModal creates an info message dialog
func InfoModal(title, message string, onDismiss func()) *tview.Modal {
	modal := tview.NewModal().
		SetText(title + "\n\n" + message).
		AddButtons([]string{"OK"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			if onDismiss != nil {
				onDismiss()
			}
		})

	modal.SetBackgroundColor(tcell.ColorDefault)
	modal.SetButtonBackgroundColor(tcell.ColorGreen)
	modal.SetButtonTextColor(tcell.ColorBlack)
	return modal
}

// InputModal creates an input dialog
func InputModal(title, label, initialValue string, onSubmit func(string), onCancel func()) tview.Primitive {
	form := tview.NewForm()

	var input string
	form.AddInputField(label, initialValue, 0, nil, func(text string) {
		input = text
	})

	form.AddButton("Submit", func() {
		if onSubmit != nil {
			onSubmit(input)
		}
	})

	form.AddButton("Cancel", func() {
		if onCancel != nil {
			onCancel()
		}
	})

	form.SetBorder(true).SetTitle(title).SetTitleAlign(tview.AlignLeft)

	// Center the form
	flex := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(form, 10, 1, true).
			AddItem(nil, 0, 1, false), 50, 1, true).
		AddItem(nil, 0, 1, false)

	return flex
}
