// SPDX-License-Identifier: Unlicense OR MIT

package router

import (
	"image"
	"reflect"
	"testing"

	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/op"
	"gioui.org/op/clip"
)

func TestKeyWakeup(t *testing.T) {
	handler := new(int)
	var ops op.Ops
	key.InputOp{Tag: handler}.Add(&ops)

	var r Router
	// Test that merely adding a handler doesn't trigger redraw.
	r.Frame(&ops)
	if _, wake := r.WakeupTime(); wake {
		t.Errorf("adding key.InputOp triggered a redraw")
	}
	// However, adding a handler queues a Focus(false) event.
	if evts := r.Events(handler); len(evts) != 1 {
		t.Errorf("no Focus event for newly registered key.InputOp")
	}
	// Verify that r.Events does trigger a redraw.
	r.Frame(&ops)
	if _, wake := r.WakeupTime(); !wake {
		t.Errorf("key.FocusEvent event didn't trigger a redraw")
	}
}

func TestKeyMultiples(t *testing.T) {
	handlers := make([]int, 3)
	ops := new(op.Ops)
	r := new(Router)

	key.SoftKeyboardOp{Show: true}.Add(ops)
	key.InputOp{Tag: &handlers[0]}.Add(ops)
	key.FocusOp{Tag: &handlers[2]}.Add(ops)
	key.InputOp{Tag: &handlers[1]}.Add(ops)

	// The last one must be focused:
	key.InputOp{Tag: &handlers[2]}.Add(ops)

	r.Frame(ops)

	assertKeyEvent(t, r.Events(&handlers[0]), false)
	assertKeyEvent(t, r.Events(&handlers[1]), false)
	assertKeyEvent(t, r.Events(&handlers[2]), true)
	assertFocus(t, r, &handlers[2])
	assertKeyboard(t, r, TextInputOpen)
}

func TestKeyStacked(t *testing.T) {
	handlers := make([]int, 4)
	ops := new(op.Ops)
	r := new(Router)

	key.InputOp{Tag: &handlers[0]}.Add(ops)
	key.FocusOp{Tag: nil}.Add(ops)
	key.SoftKeyboardOp{Show: false}.Add(ops)
	key.InputOp{Tag: &handlers[1]}.Add(ops)
	key.FocusOp{Tag: &handlers[1]}.Add(ops)
	key.InputOp{Tag: &handlers[2]}.Add(ops)
	key.SoftKeyboardOp{Show: true}.Add(ops)
	key.InputOp{Tag: &handlers[3]}.Add(ops)

	r.Frame(ops)

	assertKeyEvent(t, r.Events(&handlers[0]), false)
	assertKeyEvent(t, r.Events(&handlers[1]), true)
	assertKeyEvent(t, r.Events(&handlers[2]), false)
	assertKeyEvent(t, r.Events(&handlers[3]), false)
	assertFocus(t, r, &handlers[1])
	assertKeyboard(t, r, TextInputOpen)
}

func TestKeySoftKeyboardNoFocus(t *testing.T) {
	ops := new(op.Ops)
	r := new(Router)

	// It's possible to open the keyboard
	// without any active focus:
	key.SoftKeyboardOp{Show: true}.Add(ops)

	r.Frame(ops)

	assertFocus(t, r, nil)
	assertKeyboard(t, r, TextInputOpen)
}

func TestKeyRemoveFocus(t *testing.T) {
	handlers := make([]int, 2)
	ops := new(op.Ops)
	r := new(Router)

	// New InputOp with Focus and Keyboard:
	key.InputOp{Tag: &handlers[0]}.Add(ops)
	key.FocusOp{Tag: &handlers[0]}.Add(ops)
	key.SoftKeyboardOp{Show: true}.Add(ops)

	// New InputOp without any focus:
	key.InputOp{Tag: &handlers[1]}.Add(ops)

	r.Frame(ops)

	// Add some key events:
	event := event.Event(key.Event{Name: key.NameTab, Modifiers: key.ModShortcut, State: key.Press})
	r.Queue(event)

	assertKeyEvent(t, r.Events(&handlers[0]), true, event)
	assertKeyEvent(t, r.Events(&handlers[1]), false)
	assertFocus(t, r, &handlers[0])
	assertKeyboard(t, r, TextInputOpen)

	ops.Reset()

	// Will get the focus removed:
	key.InputOp{Tag: &handlers[0]}.Add(ops)

	// Unchanged:
	key.InputOp{Tag: &handlers[1]}.Add(ops)

	// Remove focus by focusing on a tag that don't exist.
	key.FocusOp{Tag: new(int)}.Add(ops)

	r.Frame(ops)

	assertKeyEventUnexpected(t, r.Events(&handlers[1]))
	assertFocus(t, r, nil)
	assertKeyboard(t, r, TextInputClose)

	ops.Reset()

	key.InputOp{Tag: &handlers[0]}.Add(ops)

	key.InputOp{Tag: &handlers[1]}.Add(ops)

	r.Frame(ops)

	assertKeyEventUnexpected(t, r.Events(&handlers[0]))
	assertKeyEventUnexpected(t, r.Events(&handlers[1]))
	assertFocus(t, r, nil)
	assertKeyboard(t, r, TextInputClose)

	ops.Reset()

	// Set focus to InputOp which already
	// exists in the previous frame:
	key.FocusOp{Tag: &handlers[0]}.Add(ops)
	key.InputOp{Tag: &handlers[0]}.Add(ops)
	key.SoftKeyboardOp{Show: true}.Add(ops)

	// Remove focus.
	key.InputOp{Tag: &handlers[1]}.Add(ops)
	key.FocusOp{Tag: nil}.Add(ops)

	r.Frame(ops)

	assertKeyEventUnexpected(t, r.Events(&handlers[1]))
	assertFocus(t, r, nil)
	assertKeyboard(t, r, TextInputOpen)
}

func TestKeyFocusedInvisible(t *testing.T) {
	handlers := make([]int, 2)
	ops := new(op.Ops)
	r := new(Router)

	// Set new InputOp with focus:
	key.FocusOp{Tag: &handlers[0]}.Add(ops)
	key.InputOp{Tag: &handlers[0]}.Add(ops)
	key.SoftKeyboardOp{Show: true}.Add(ops)

	// Set new InputOp without focus:
	key.InputOp{Tag: &handlers[1]}.Add(ops)

	r.Frame(ops)

	assertKeyEvent(t, r.Events(&handlers[0]), true)
	assertKeyEvent(t, r.Events(&handlers[1]), false)
	assertFocus(t, r, &handlers[0])
	assertKeyboard(t, r, TextInputOpen)

	ops.Reset()

	//
	// Removed first (focused) element!
	//

	// Unchanged:
	key.InputOp{Tag: &handlers[1]}.Add(ops)

	r.Frame(ops)

	assertKeyEventUnexpected(t, r.Events(&handlers[0]))
	assertKeyEventUnexpected(t, r.Events(&handlers[1]))
	assertFocus(t, r, nil)
	assertKeyboard(t, r, TextInputClose)

	ops.Reset()

	// Respawn the first element:
	// It must receive one `Event{Focus: false}`.
	key.InputOp{Tag: &handlers[0]}.Add(ops)

	// Unchanged
	key.InputOp{Tag: &handlers[1]}.Add(ops)

	r.Frame(ops)

	assertKeyEvent(t, r.Events(&handlers[0]), false)
	assertKeyEventUnexpected(t, r.Events(&handlers[1]))
	assertFocus(t, r, nil)
	assertKeyboard(t, r, TextInputClose)

}

func TestNoOps(t *testing.T) {
	r := new(Router)
	r.Frame(nil)
}

func TestTabFocus(t *testing.T) {
	handlers := make([]int, 3)
	ops := new(op.Ops)
	r := new(Router)

	for i := range handlers {
		key.InputOp{Tag: &handlers[i]}.Add(ops)
	}
	r.Frame(ops)

	tab := func(mod key.Modifiers) {
		r.Queue(
			key.Event{Name: key.NameTab, State: key.Press, Modifiers: mod},
			key.Event{Name: key.NameTab, State: key.Release, Modifiers: mod},
		)
	}
	tab(0)
	tab(key.ModShift)
	assertFocus(t, r, &handlers[2])
}

func TestDirectionalFocus(t *testing.T) {
	ops := new(op.Ops)
	r := new(Router)
	handlers := []image.Rectangle{
		image.Rect(10, 10, 50, 50),
		image.Rect(50, 20, 100, 80),
		image.Rect(20, 26, 60, 80),
		image.Rect(10, 60, 50, 100),
	}

	for i, bounds := range handlers {
		cl := clip.Rect(bounds).Push(ops)
		key.InputOp{Tag: &handlers[i]}.Add(ops)
		cl.Pop()
	}
	r.Frame(ops)

	r.MoveFocus(FocusLeft)
	assertFocus(t, r, &handlers[0])
	r.MoveFocus(FocusLeft)
	assertFocus(t, r, &handlers[0])
	r.MoveFocus(FocusRight)
	assertFocus(t, r, &handlers[1])
	r.MoveFocus(FocusRight)
	assertFocus(t, r, &handlers[1])
	r.MoveFocus(FocusDown)
	assertFocus(t, r, &handlers[2])
	r.MoveFocus(FocusDown)
	assertFocus(t, r, &handlers[2])
	r.MoveFocus(FocusLeft)
	assertFocus(t, r, &handlers[3])
	r.MoveFocus(FocusUp)
	assertFocus(t, r, &handlers[0])
}

func assertKeyEvent(t *testing.T, events []event.Event, expected bool, expectedInputs ...event.Event) {
	t.Helper()
	var evtFocus int
	var evtKeyPress int
	for _, e := range events {
		switch ev := e.(type) {
		case key.FocusEvent:
			if ev.Focus != expected {
				t.Errorf("focus is expected to be %v, got %v", expected, ev.Focus)
			}
			evtFocus++
		case key.Event, key.EditEvent:
			if len(expectedInputs) <= evtKeyPress {
				t.Errorf("unexpected key events")
			}
			if !reflect.DeepEqual(ev, expectedInputs[evtKeyPress]) {
				t.Errorf("expected %v events, got %v", expectedInputs[evtKeyPress], ev)
			}
			evtKeyPress++
		}
	}
	if evtFocus <= 0 {
		t.Errorf("expected focus event")
	}
	if evtFocus > 1 {
		t.Errorf("expected single focus event")
	}
	if evtKeyPress != len(expectedInputs) {
		t.Errorf("expected key events")
	}
}

func assertKeyEventUnexpected(t *testing.T, events []event.Event) {
	t.Helper()
	var evtFocus int
	for _, e := range events {
		switch e.(type) {
		case key.FocusEvent:
			evtFocus++
		}
	}
	if evtFocus > 1 {
		t.Errorf("unexpected focus event")
	}
}

func assertFocus(t *testing.T, router *Router, expected event.Tag) {
	t.Helper()
	if got := router.key.queue.focus; got != expected {
		t.Errorf("expected %v to be focused, got %v", expected, got)
	}
}

func assertKeyboard(t *testing.T, router *Router, expected TextInputState) {
	t.Helper()
	if got := router.key.queue.state; got != expected {
		t.Errorf("expected %v keyboard, got %v", expected, got)
	}
}
