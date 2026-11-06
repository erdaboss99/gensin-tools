package control

import "syscall"

var (
	user32 = syscall.NewLazyDLL("user32.dll")
	gdi32  = syscall.NewLazyDLL("gdi32.dll")

	procEnumWindows              = user32.NewProc("EnumWindows")
	procGetWindowThreadProcessID = user32.NewProc("GetWindowThreadProcessId")
	procIsWindowVisible          = user32.NewProc("IsWindowVisible")
	procIsIconic                 = user32.NewProc("IsIconic")
	procShowWindow               = user32.NewProc("ShowWindow")
	procSetForegroundWindow      = user32.NewProc("SetForegroundWindow")
	procGetForegroundWindow      = user32.NewProc("GetForegroundWindow")
	procGetClientRect            = user32.NewProc("GetClientRect")
	procClientToScreen           = user32.NewProc("ClientToScreen")
	procSetCursorPos             = user32.NewProc("SetCursorPos")
	procSendInput                = user32.NewProc("SendInput")
	procGetDC                    = user32.NewProc("GetDC")
	procReleaseDC                = user32.NewProc("ReleaseDC")
	procSetProcessDPIAware       = user32.NewProc("SetProcessDPIAware")

	procCreateCompatibleDC     = gdi32.NewProc("CreateCompatibleDC")
	procCreateCompatibleBitmap = gdi32.NewProc("CreateCompatibleBitmap")
	procSelectObject           = gdi32.NewProc("SelectObject")
	procBitBlt                 = gdi32.NewProc("BitBlt")
	procGetDIBits              = gdi32.NewProc("GetDIBits")
	procDeleteObject           = gdi32.NewProc("DeleteObject")
	procDeleteDC               = gdi32.NewProc("DeleteDC")
)

const (
	swRestore = 9

	srcCopy    = 0x00CC0020
	captureBlt = 0x40000000
	dibRGB     = 0
	biRGB      = 0

	tokenElevation = 20

	inputMouseKind    = 0
	inputKeyboardKind = 1

	mouseMove     = 0x0001
	mouseLeftDown = 0x0002
	mouseLeftUp   = 0x0004
	mouseWheel    = 0x0800
	wheelDelta    = 120

	keyUp      = 0x0002
	keyUnicode = 0x0004
	vkReturn   = 0x0D
)

type point struct{ x, y int32 }

type rect struct{ left, top, right, bottom int32 }

type bitmapInfoHeader struct {
	size          uint32
	width, height int32
	planes        uint16
	bitCount      uint16
	compression   uint32
	sizeImage     uint32
	xPPM, yPPM    int32
	clrUsed       uint32
	clrImportant  uint32
}

// mouseInput and keyboardInput mirror the Win32 INPUT structure on 64-bit Windows:
// a 32-bit type, padding, then a 32-byte union aligned to 8 bytes, for 40 bytes in
// total. SendInput rejects any other size. See TestInputStructLayout.
type mouseInput struct {
	kind      uint32
	_         uint32
	dx, dy    int32
	mouseData uint32
	flags     uint32
	time      uint32
	extra     uintptr
}

type keyboardInput struct {
	kind  uint32
	_     uint32
	vk    uint16
	scan  uint16
	flags uint32
	time  uint32
	extra uintptr
	_     [8]byte
}
