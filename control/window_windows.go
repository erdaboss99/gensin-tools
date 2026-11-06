package control

import (
	"context"
	"fmt"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

const (
	focusPollDelay     = 200 * time.Millisecond
	windowRestoreDelay = 500 * time.Millisecond
	cursorMoveDelay    = 40 * time.Millisecond
	buttonHoldDelay    = 20 * time.Millisecond
	inputStepDelay     = 10 * time.Millisecond
)

// gameExecutables lists the global and Chinese client process names.
var gameExecutables = []string{"genshinimpact.exe", "yuanshen.exe"}

type window struct {
	hwnd          uintptr
	origin        point
	width, height int
}

func isElevated() bool {
	process, err := syscall.GetCurrentProcess()
	if err != nil {
		return false
	}
	var token syscall.Token
	if err := syscall.OpenProcessToken(process, syscall.TOKEN_QUERY, &token); err != nil {
		return false
	}
	defer token.Close()
	var elevated, size uint32
	err = syscall.GetTokenInformation(token, tokenElevation, (*byte)(unsafe.Pointer(&elevated)), uint32(unsafe.Sizeof(elevated)), &size)
	return err == nil && elevated != 0
}

func gamePIDs() (map[uint32]bool, error) {
	snapshot, err := syscall.CreateToolhelp32Snapshot(syscall.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return nil, fmt.Errorf("list processes: %w", err)
	}
	defer syscall.CloseHandle(snapshot)

	pids := make(map[uint32]bool)
	var entry syscall.ProcessEntry32
	entry.Size = uint32(unsafe.Sizeof(entry))
	for err = syscall.Process32First(snapshot, &entry); err == nil; err = syscall.Process32Next(snapshot, &entry) {
		name := strings.ToLower(syscall.UTF16ToString(entry.ExeFile[:]))
		for _, exe := range gameExecutables {
			if name == exe {
				pids[entry.ProcessID] = true
			}
		}
	}
	return pids, nil
}

func findGameWindow() (uintptr, error) {
	pids, err := gamePIDs()
	if err != nil {
		return 0, err
	}
	if len(pids) == 0 {
		return 0, ErrNotRunning
	}
	// The game can own several visible windows, such as launcher or helper
	// surfaces, so the main one is identified as the largest. A minimized main
	// window has no client area, which leaves it as the fallback.
	var found uintptr
	largest := -1
	callback := syscall.NewCallback(func(hwnd, _ uintptr) uintptr {
		var pid uint32
		procGetWindowThreadProcessID.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
		if visible, _, _ := procIsWindowVisible.Call(hwnd); visible == 0 || !pids[pid] {
			return 1
		}
		area := 0
		if _, w, h, err := clientArea(hwnd); err == nil {
			area = w * h
		}
		if area > largest {
			found, largest = hwnd, area
		}
		return 1
	})
	procEnumWindows.Call(callback, 0)
	if found == 0 {
		return 0, ErrNotRunning
	}
	return found, nil
}

func clientArea(hwnd uintptr) (point, int, int, error) {
	var r rect
	if ok, _, err := procGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&r))); ok == 0 {
		return point{}, 0, 0, fmt.Errorf("GetClientRect: %w", err)
	}
	var origin point
	if ok, _, err := procClientToScreen.Call(hwnd, uintptr(unsafe.Pointer(&origin))); ok == 0 {
		return point{}, 0, 0, fmt.Errorf("ClientToScreen: %w", err)
	}
	return origin, int(r.right - r.left), int(r.bottom - r.top), nil
}

// Probe reports on the environment without changing anything.
func Probe() Status {
	procSetProcessDPIAware.Call()
	status := Status{Elevated: isElevated()}
	hwnd, err := findGameWindow()
	if err != nil {
		return status
	}
	status.GameRunning = true
	if iconic, _, _ := procIsIconic.Call(hwnd); iconic != 0 {
		status.Minimized = true
		return status
	}
	if _, w, h, err := clientArea(hwnd); err == nil {
		status.Width, status.Height = w, h
	}
	return status
}

// Attach finds the game and brings it to the foreground. It waits for focus until
// ctx ends, so the user can click on the game if Windows refuses the request.
func Attach(ctx context.Context, opts AttachOptions) (Game, error) {
	// Without DPI awareness Windows would report and accept virtualized
	// coordinates on scaled displays, misplacing every click and crop.
	procSetProcessDPIAware.Call()

	if opts.NeedInput && !isElevated() {
		return nil, ErrNotElevated
	}
	hwnd, err := findGameWindow()
	if err != nil {
		return nil, err
	}
	if iconic, _, _ := procIsIconic.Call(hwnd); iconic != 0 {
		procShowWindow.Call(hwnd, swRestore)
	}
	procSetForegroundWindow.Call(hwnd)

	win := &window{hwnd: hwnd}
	if !win.active() {
		fmt.Println("Waiting for the game to take focus. Please click on the game window.")
	}
	for !win.active() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(focusPollDelay):
		}
	}

	// A window that was just restored is still resizing and redrawing.
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(windowRestoreDelay):
	}

	win.origin, win.width, win.height, err = clientArea(hwnd)
	if err != nil {
		return nil, err
	}
	return win, nil
}

func (w *window) active() bool {
	foreground, _, _ := procGetForegroundWindow.Call()
	return foreground == w.hwnd
}

func (w *window) checkFocus() error {
	if !w.active() {
		return ErrFocusLost
	}
	return nil
}

func (w *window) Size() (int, int) { return w.width, w.height }
