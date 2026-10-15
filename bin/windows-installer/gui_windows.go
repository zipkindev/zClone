package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"unsafe"
)

var (
	user32       = syscall.NewLazyDLL("user32.dll")
	kernel32     = syscall.NewLazyDLL("kernel32.dll")
	shell32      = syscall.NewLazyDLL("shell32.dll")
	ole32        = syscall.NewLazyDLL("ole32.dll")
	createWindow = user32.NewProc("CreateWindowExW")
	sendMessage  = user32.NewProc("SendMessageW")
	setText      = user32.NewProc("SetWindowTextW")
	enableWindow = user32.NewProc("EnableWindow")
	defaultProc  = user32.NewProc("DefWindowProcW")
	activeWindow *updaterWindow
)

const (
	controlFolder   = 101
	controlBrowse   = 102
	controlInstall  = 103
	controlClose    = 104
	controlStatus   = 105
	controlProgress = 106
	progressMessage = 0x8001
)

type windowClass struct {
	Size, Style                        uint32
	Procedure                          uintptr
	ClassExtra, WindowExtra            int32
	Instance, Icon, Cursor, Background uintptr
	Menu, Name                         *uint16
	SmallIcon                          uintptr
}

type windowMessage struct {
	Window         uintptr
	Message        uint32
	WParam, LParam uintptr
	Time           uint32
	X, Y           int32
	Private        uint32
}

type updaterWindow struct {
	window, folder, browse, install, close, status, bar uintptr
	options                                             updateOptions
	mu                                                  sync.Mutex
	busy                                                bool
	text                                                string
	percent                                             int
	pendingError                                        string
}

func wide(value string) *uint16 { p, _ := syscall.UTF16PtrFromString(value); return p }
func textOf(window uintptr) string {
	size, _, _ := user32.NewProc("GetWindowTextLengthW").Call(window)
	buffer := make([]uint16, size+1)
	user32.NewProc("GetWindowTextW").Call(window, uintptr(unsafe.Pointer(&buffer[0])), uintptr(len(buffer)))
	return syscall.UTF16ToString(buffer)
}
func putText(window uintptr, value string) {
	setText.Call(window, uintptr(unsafe.Pointer(wide(value))))
}

func (w *updaterWindow) notify(text string, percent int) {
	w.mu.Lock()
	w.text = text
	w.percent = percent
	w.mu.Unlock()
	user32.NewProc("PostMessageW").Call(w.window, progressMessage, 0, 0)
}

func (w *updaterWindow) display() {
	w.mu.Lock()
	text, percent, busy := w.text, w.percent, w.busy
	failure := w.pendingError
	w.pendingError = ""
	w.mu.Unlock()
	putText(w.status, text)
	sendMessage.Call(w.bar, 0x40a, boolValue(percent < 0), 35)
	if percent >= 0 {
		sendMessage.Call(w.bar, 0x402, uintptr(percent), 0)
	}
	for _, control := range []uintptr{w.folder, w.browse, w.install, w.close} {
		enableWindow.Call(control, boolValue(!busy))
	}
	if failure != "" {
		user32.NewProc("MessageBoxW").Call(w.window, uintptr(unsafe.Pointer(wide(failure))), uintptr(unsafe.Pointer(wide("zClone Updater"))), 0x10)
	}
}
func boolValue(value bool) uintptr {
	if value {
		return 1
	}
	return 0
}

func (w *updaterWindow) start() {
	w.mu.Lock()
	if w.busy {
		w.mu.Unlock()
		return
	}
	w.mu.Unlock()
	folder := strings.TrimSpace(textOf(w.folder))
	if folder == "" {
		w.notify("Choose an installation folder first.", 0)
		return
	}
	w.options.Directory = folder
	w.mu.Lock()
	w.busy = true
	w.mu.Unlock()
	w.notify("Preparing…", 0)
	w.display()
	options := w.options
	go func() {
		result, err := performUpdate(options, w.notify)
		message := "Installed successfully.\r\n" + result.Path
		if result.Warning != "" {
			message += "\r\n" + result.Warning
		}
		percent := 100
		if err != nil {
			message = "Installation failed: " + err.Error()
			percent = 0
		}
		w.mu.Lock()
		w.busy = false
		w.text = message
		w.percent = percent
		if err != nil {
			w.pendingError = err.Error()
		}
		w.mu.Unlock()
		user32.NewProc("PostMessageW").Call(w.window, progressMessage, 0, 0)
	}()
}

func chooseFolder(owner uintptr, current string) string {
	type browseInfo struct {
		Owner, Root         uintptr
		Display, Title      *uint16
		Flags               uint32
		Callback, Parameter uintptr
		Image               int32
	}
	buffer := make([]uint16, 32768)
	initial := wide(current)
	callback := syscall.NewCallback(func(window uintptr, message uint32, lparam, param uintptr) uintptr {
		if message == 1 {
			sendMessage.Call(window, 0x467, 1, param)
		}
		return 0
	})
	info := browseInfo{Owner: owner, Display: &buffer[0], Title: wide("Choose the zClone installation folder"), Flags: 0x51, Callback: callback, Parameter: uintptr(unsafe.Pointer(initial))}
	item, _, _ := shell32.NewProc("SHBrowseForFolderW").Call(uintptr(unsafe.Pointer(&info)))
	runtime.KeepAlive(initial)
	if item == 0 {
		return ""
	}
	defer ole32.NewProc("CoTaskMemFree").Call(item)
	ok, _, _ := shell32.NewProc("SHGetPathFromIDListW").Call(item, uintptr(unsafe.Pointer(&buffer[0])))
	if ok == 0 {
		return ""
	}
	return syscall.UTF16ToString(buffer)
}

func windowProcedure(window uintptr, message uint32, wparam, lparam uintptr) uintptr {
	w := activeWindow
	if w != nil {
		switch message {
		case 0x111:
			switch int(wparam & 0xffff) {
			case controlBrowse:
				if folder := chooseFolder(window, textOf(w.folder)); folder != "" {
					putText(w.folder, folder)
				}
			case controlInstall:
				w.start()
			case controlClose:
				user32.NewProc("SendMessageW").Call(window, 0x10, 0, 0)
			}
			return 0
		case progressMessage:
			w.display()
			return 0
		case 0x10:
			w.mu.Lock()
			busy := w.busy
			w.mu.Unlock()
			if busy {
				return 0
			}
			user32.NewProc("DestroyWindow").Call(window)
			return 0
		case 0x2:
			user32.NewProc("PostQuitMessage").Call(0)
			return 0
		}
	}
	result, _, _ := defaultProc.Call(window, uintptr(message), wparam, lparam)
	return result
}

func runGUI(options updateOptions) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	initialized, _, _ := ole32.NewProc("CoInitializeEx").Call(0, 2)
	if int32(initialized) < 0 {
		return fmt.Errorf("initialize folder picker: 0x%x", initialized)
	}
	defer ole32.NewProc("CoUninitialize").Call()
	user32.NewProc("SetProcessDPIAware").Call()
	controls := struct{ Size, Classes uint32 }{8, 0x20}
	syscall.NewLazyDLL("comctl32.dll").NewProc("InitCommonControlsEx").Call(uintptr(unsafe.Pointer(&controls)))
	instance, _, _ := kernel32.NewProc("GetModuleHandleW").Call(0)
	cursor, _, _ := user32.NewProc("LoadCursorW").Call(0, 32512)
	class := windowClass{Instance: instance, Name: wide("zCloneUpdaterWindow"), Cursor: cursor, Background: 16, Procedure: syscall.NewCallback(windowProcedure)}
	class.Size = uint32(unsafe.Sizeof(class))
	atom, _, err := user32.NewProc("RegisterClassExW").Call(uintptr(unsafe.Pointer(&class)))
	if atom == 0 {
		return fmt.Errorf("create updater window: %w", err)
	}
	defer func() {
		user32.NewProc("UnregisterClassW").Call(uintptr(unsafe.Pointer(class.Name)), instance)
		runtime.KeepAlive(class.Name)
	}()
	w := &updaterWindow{options: options}
	activeWindow = w
	defer func() { activeWindow = nil }()
	dpi := uintptr(96)
	if proc := user32.NewProc("GetDpiForSystem"); proc.Find() == nil {
		dpi, _, _ = proc.Call()
	}
	scale := func(value int) uintptr { return uintptr(value) * dpi / 96 }
	w.window, _, err = createWindow.Call(0, uintptr(unsafe.Pointer(class.Name)), uintptr(unsafe.Pointer(wide("zClone Updater"))), 0x00ca0000, 0x80000000, 0x80000000, scale(600), scale(370), 0, 0, instance, 0)
	if w.window == 0 {
		return fmt.Errorf("create updater window: %w", err)
	}
	font, _, _ := syscall.NewLazyDLL("gdi32.dll").NewProc("GetStockObject").Call(17)
	add := func(kind, text string, style uintptr, id, x, y, width, height int) uintptr {
		control, _, _ := createWindow.Call(0, uintptr(unsafe.Pointer(wide(kind))), uintptr(unsafe.Pointer(wide(text))), 0x50000000|style, scale(x), scale(y), scale(width), scale(height), w.window, uintptr(id), instance, 0)
		sendMessage.Call(control, 0x30, font, 1)
		return control
	}
	add("STATIC", "Install or update zClone", 0, 0, 22, 18, 550, 25)
	add("STATIC", "Choose a folder, then click Install / Update. Your files and settings stay in place.", 0, 0, 22, 50, 550, 35)
	add("STATIC", "Installation folder", 0, 0, 22, 90, 550, 20)
	w.folder = add("EDIT", options.Directory, 0x00810080, controlFolder, 22, 114, 444, 27)
	w.browse = add("BUTTON", "Browse…", 0x10000, controlBrowse, 478, 114, 90, 27)
	w.bar = add("msctls_progress32", "", 8, controlProgress, 22, 163, 546, 22)
	sendMessage.Call(w.bar, 0x406, 0, 100)
	w.status = add("STATIC", "Ready. The latest stable release will be downloaded and installed here.", 0, controlStatus, 22, 199, 546, 65)
	w.install = add("BUTTON", "Install / Update", 0x10001, controlInstall, 322, 283, 150, 30)
	w.close = add("BUTTON", "Close", 0x10000, controlClose, 484, 283, 84, 30)
	for _, handle := range []uintptr{w.folder, w.browse, w.bar, w.status, w.install, w.close} {
		if handle == 0 {
			user32.NewProc("DestroyWindow").Call(w.window)
			return fmt.Errorf("create updater controls")
		}
	}
	user32.NewProc("ShowWindow").Call(w.window, 1)
	user32.NewProc("UpdateWindow").Call(w.window)
	var msg windowMessage
	for {
		result, _, err := user32.NewProc("GetMessageW").Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if int32(result) == -1 {
			return err
		}
		if result == 0 {
			break
		}
		dialog, _, _ := user32.NewProc("IsDialogMessageW").Call(w.window, uintptr(unsafe.Pointer(&msg)))
		if dialog == 0 {
			user32.NewProc("TranslateMessage").Call(uintptr(unsafe.Pointer(&msg)))
			user32.NewProc("DispatchMessageW").Call(uintptr(unsafe.Pointer(&msg)))
		}
	}
	return nil
}

func prepareCLI() {
	if handle, err := syscall.GetStdHandle(syscall.STD_OUTPUT_HANDLE); err == nil && handle != syscall.InvalidHandle && handle != 0 {
		return
	}
	kernel32.NewProc("AttachConsole").Call(0xffffffff)
	if handle, err := syscall.GetStdHandle(syscall.STD_OUTPUT_HANDLE); err == nil {
		os.Stdout = os.NewFile(uintptr(handle), "stdout")
	}
	if handle, err := syscall.GetStdHandle(syscall.STD_ERROR_HANDLE); err == nil {
		os.Stderr = os.NewFile(uintptr(handle), "stderr")
	}
}

func hideConsole(command *exec.Cmd) {
	command.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
}
