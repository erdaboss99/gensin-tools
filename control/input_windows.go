package control

import (
	"errors"
	"fmt"
	"time"
	"unicode/utf16"
	"unsafe"
)

func (w *window) MoveTo(x, y int) error {
	if err := w.checkFocus(); err != nil {
		return err
	}
	ok, _, err := procSetCursorPos.Call(uintptr(int(w.origin.x)+x), uintptr(int(w.origin.y)+y))
	if ok == 0 {
		return fmt.Errorf("SetCursorPos: %w", err)
	}
	return nil
}

func (w *window) Click(x, y int) error {
	if err := w.MoveTo(x, y); err != nil {
		return err
	}
	// The game polls input once per frame, so a press and release in the same
	// frame could be missed entirely.
	time.Sleep(cursorMoveDelay)
	if err := sendMouse(mouseInput{flags: mouseLeftDown}); err != nil {
		return err
	}
	time.Sleep(buttonHoldDelay)
	return sendMouse(mouseInput{flags: mouseLeftUp})
}

func (w *window) Scroll(ticks int) error {
	step, delta := 1, int32(wheelDelta)
	if ticks < 0 {
		step, delta = -1, -wheelDelta
	}
	for i := 0; i != ticks; i += step {
		if err := w.checkFocus(); err != nil {
			return err
		}
		if err := sendMouse(mouseInput{flags: mouseWheel, mouseData: uint32(delta)}); err != nil {
			return err
		}
		time.Sleep(inputStepDelay)
	}
	return nil
}

func (w *window) Type(text string) error {
	for _, unit := range utf16.Encode([]rune(text)) {
		if err := w.checkFocus(); err != nil {
			return err
		}
		if err := sendKey(keyboardInput{scan: unit, flags: keyUnicode}); err != nil {
			return err
		}
		if err := sendKey(keyboardInput{scan: unit, flags: keyUnicode | keyUp}); err != nil {
			return err
		}
		time.Sleep(inputStepDelay)
	}
	return nil
}

func (w *window) PressEnter() error {
	if err := w.checkFocus(); err != nil {
		return err
	}
	if err := sendKey(keyboardInput{vk: vkReturn}); err != nil {
		return err
	}
	return sendKey(keyboardInput{vk: vkReturn, flags: keyUp})
}

func sendMouse(input mouseInput) error {
	input.kind = inputMouseKind
	return sendInput(unsafe.Pointer(&input))
}

func sendKey(input keyboardInput) error {
	input.kind = inputKeyboardKind
	return sendInput(unsafe.Pointer(&input))
}

func sendInput(input unsafe.Pointer) error {
	sent, _, err := procSendInput.Call(1, uintptr(input), unsafe.Sizeof(mouseInput{}))
	if sent != 1 {
		return errors.Join(errors.New("SendInput was blocked"), err)
	}
	return nil
}
