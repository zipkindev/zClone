"""Drive the updater's native Windows controls using disposable release fixtures."""
import ctypes
from ctypes import wintypes
import hashlib
import json
import os
from pathlib import Path
import subprocess
import struct
import zipfile
import time


def exercise_gui(updater, archive, root, screenshot=None):
    root.mkdir()
    user = ctypes.WinDLL('user32', use_last_error=True)
    user.GetDlgItem.argtypes = [wintypes.HWND, ctypes.c_int]
    user.GetDlgItem.restype = wintypes.HWND
    user.SendMessageW.argtypes = [wintypes.HWND, wintypes.UINT, wintypes.WPARAM, wintypes.LPARAM]
    user.SendMessageW.restype = wintypes.LPARAM
    user.PostMessageW.argtypes = user.SendMessageW.argtypes
    user.PostMessageW.restype = wintypes.BOOL
    user.GetWindowThreadProcessId.argtypes = [wintypes.HWND, ctypes.POINTER(wintypes.DWORD)]
    user.GetClassNameW.argtypes = [wintypes.HWND, wintypes.LPWSTR, ctypes.c_int]
    user.GetWindowTextW.argtypes = [wintypes.HWND, wintypes.LPWSTR, ctypes.c_int]
    user.IsWindow.argtypes = [wintypes.HWND]
    user.IsWindowEnabled.argtypes = [wintypes.HWND]
    user.IsWindowVisible.argtypes = [wintypes.HWND]
    callback_type = ctypes.WINFUNCTYPE(wintypes.BOOL, wintypes.HWND, wintypes.LPARAM)
    user.EnumWindows.argtypes = [callback_type, wintypes.LPARAM]
    user.EnumChildWindows.argtypes = [wintypes.HWND, callback_type, wintypes.LPARAM]
    destination = root / 'chosen folder with spaces'
    destination.mkdir()
    env = dict(os.environ, LOCALAPPDATA=str(root / 'preferences'))
    command = [str(updater), '--gui', '--archive', str(archive), '--sha256',
               hashlib.sha256(archive.read_bytes()).hexdigest()]
    process = subprocess.Popen(command, env=env, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)

    def windows(kind):
        found = []
        @callback_type
        def visit(window, parameter):
            pid = wintypes.DWORD()
            user.GetWindowThreadProcessId(window, ctypes.byref(pid))
            name = ctypes.create_unicode_buffer(256)
            user.GetClassNameW(window, name, len(name))
            if pid.value == process.pid and name.value == kind and user.IsWindowVisible(window):
                found.append(window)
            return True
        user.EnumWindows(visit, 0)
        return found

    def wait(predicate, timeout=30):
        deadline = time.monotonic() + timeout
        while time.monotonic() < deadline:
            value = predicate()
            if value:
                return value
            if process.poll() is not None:
                raise AssertionError('Updater GUI exited unexpectedly: ' + str(process.returncode))
            time.sleep(0.05)
        raise AssertionError('Timed out waiting for updater GUI')

    def text(control):
        buffer = ctypes.create_unicode_buffer(4096)
        user.SendMessageW(control, 0xD, len(buffer), ctypes.addressof(buffer))
        return buffer.value

    try:
        window = wait(lambda: windows('zCloneUpdaterWindow'))[0]
        folder = user.GetDlgItem(window, 101)
        assert folder and text(folder)
        assert not (destination / 'zclone.exe').exists(), 'GUI installed before user confirmation'
        # Open the real shell picker and select an existing disposable folder.
        user.PostMessageW(window, 0x111, 102, 0)
        picker = wait(lambda: windows('#32770'))[0]
        path = ctypes.create_unicode_buffer(str(destination))
        edits = []
        @callback_type
        def find_edit(control, parameter):
            name = ctypes.create_unicode_buffer(256)
            user.GetClassNameW(control, name, len(name))
            if name.value == 'Edit':
                edits.append(control)
            return True
        user.EnumChildWindows(picker, find_edit, 0)
        assert edits, 'Folder picker has no path entry field'
        # This shell message carries a pointer, so the string must live in the
        # child process. Allocate only a writable data buffer in our own updater.
        kernel = ctypes.WinDLL('kernel32', use_last_error=True)
        kernel.VirtualAllocEx.argtypes = [wintypes.HANDLE, ctypes.c_void_p, ctypes.c_size_t,
                                         wintypes.DWORD, wintypes.DWORD]
        kernel.VirtualAllocEx.restype = ctypes.c_void_p
        kernel.WriteProcessMemory.argtypes = [wintypes.HANDLE, ctypes.c_void_p, ctypes.c_void_p,
                                             ctypes.c_size_t, ctypes.POINTER(ctypes.c_size_t)]
        kernel.VirtualFreeEx.argtypes = [wintypes.HANDLE, ctypes.c_void_p, ctypes.c_size_t, wintypes.DWORD]
        handle = wintypes.HANDLE(int(process._handle))
        remote = kernel.VirtualAllocEx(handle, None, ctypes.sizeof(path), 0x3000, 4)
        if not remote:
            raise ctypes.WinError(ctypes.get_last_error())
        try:
            written = ctypes.c_size_t()
            if not kernel.WriteProcessMemory(handle, remote, path, ctypes.sizeof(path), ctypes.byref(written)):
                raise ctypes.WinError(ctypes.get_last_error())
            assert written.value == ctypes.sizeof(path)
            selection_result = user.SendMessageW(picker, 0x467, 1, remote)
            time.sleep(0.3)
            wait(lambda: user.IsWindowEnabled(user.GetDlgItem(picker, 1)))
            diagnostic = {'expected': str(destination), 'selection_result': selection_result,
                          'picker_text': text(edits[0]), 'initial_folder': text(folder)}
            if screenshot:
                capture_window(user, picker, screenshot.with_name('updater-picker.bmp'))
            user.PostMessageW(user.GetDlgItem(picker, 1), 0xF5, 0, 0)
            wait(lambda: not user.IsWindow(picker))
        finally:
            kernel.VirtualFreeEx(handle, remote, 0, 0x8000)
        time.sleep(0.1)
        diagnostic['returned_folder'] = text(folder)
        print('FOLDER_SELECTION=' + json.dumps(diagnostic), flush=True)
        if screenshot:
            capture_window(user, window, screenshot.with_name('updater-selection.bmp'))
        wait(lambda: Path(text(folder)).samefile(destination))
        # Verify cancelling the picker retains the selection.
        user.PostMessageW(window, 0x111, 102, 0)
        picker = wait(lambda: windows('#32770'))[0]
        user.PostMessageW(user.GetDlgItem(picker, 2), 0xF5, 0, 0)
        wait(lambda: not user.IsWindow(picker))
        assert Path(text(folder)).samefile(destination)
        user.PostMessageW(window, 0x111, 103, 0)
        status = user.GetDlgItem(window, 105)
        wait(lambda: 'Installed successfully.' in text(status), timeout=180)
        wait(lambda: user.SendMessageW(user.GetDlgItem(window, 106), 0x408, 0, 0) == 100)
        if screenshot:
            capture_window(user, window, screenshot)
        with zipfile.ZipFile(archive) as package:
            original = package.read(next(n for n in package.namelist() if n.endswith('/zclone.exe')))
        binary = destination / 'zclone.exe'
        assert hashlib.sha256(binary.read_bytes()).digest() == hashlib.sha256(original).digest()
        from windows_installer_qa import allocated_bytes
        assert allocated_bytes(binary) < binary.stat().st_size
        saved = json.loads((root / 'preferences/zclone-updater/settings.json').read_text())
        assert Path(saved['install_directory']).samefile(destination)
        user.PostMessageW(window, 0x10, 0, 0)
        assert process.wait(timeout=15) == 0
        # Double-click behavior uses the remembered folder without flags.
        process = subprocess.Popen([str(updater)], env=env, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
        window = wait(lambda: windows('zCloneUpdaterWindow'))[0]
        assert Path(text(user.GetDlgItem(window, 101))).samefile(destination)
        user.PostMessageW(window, 0x10, 0, 0)
        assert process.wait(timeout=15) == 0
        # A GUI verification error must be visible and preserve the installed image.
        failed = command.copy()
        failed[failed.index('--sha256') + 1] = '0' * 64
        failed += ['--install-dir', str(destination)]
        process = subprocess.Popen(failed, env=env, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
        window = wait(lambda: windows('zCloneUpdaterWindow'))[0]
        user.PostMessageW(window, 0x111, 103, 0)
        wait(lambda: 'Installation failed:' in text(user.GetDlgItem(window, 105)))
        dialog = wait(lambda: windows('#32770'))[0]
        user.PostMessageW(user.GetDlgItem(dialog, 1), 0xF5, 0, 0)
        wait(lambda: not user.IsWindow(dialog))
        assert hashlib.sha256(binary.read_bytes()).digest() == hashlib.sha256(original).digest()
        assert user.IsWindowEnabled(user.GetDlgItem(window, 103))
        user.PostMessageW(window, 0x10, 0, 0)
        assert process.wait(timeout=15) == 0
        return {'passed': True, 'tests': ['native window', 'folder picker selection/cancel',
                 'installation progress', 'successful install', 'native compression',
                 'remembered destination', 'double-click opens GUI', 'close without installing',
                 'visible checksum failure', 'failed install preserves application']}
    finally:
        if process.poll() is None:
            process.kill()
            process.wait(timeout=15)


def capture_window(user, window, output):
    gdi = ctypes.WinDLL('gdi32', use_last_error=True)
    rectangle = wintypes.RECT()
    user.GetWindowRect.argtypes = [wintypes.HWND, ctypes.POINTER(wintypes.RECT)]
    user.GetWindowRect(window, ctypes.byref(rectangle))
    width, height = rectangle.right - rectangle.left, rectangle.bottom - rectangle.top
    assert width > 0 and height > 0
    gdi.CreateCompatibleDC.argtypes = [wintypes.HDC]
    gdi.CreateCompatibleDC.restype = wintypes.HDC
    gdi.CreateDIBSection.argtypes = [wintypes.HDC, ctypes.c_void_p, wintypes.UINT,
                                   ctypes.POINTER(ctypes.c_void_p), wintypes.HANDLE, wintypes.DWORD]
    gdi.CreateDIBSection.restype = wintypes.HANDLE
    gdi.SelectObject.argtypes = [wintypes.HDC, wintypes.HANDLE]
    gdi.SelectObject.restype = wintypes.HANDLE
    gdi.DeleteObject.argtypes = [wintypes.HANDLE]
    gdi.DeleteDC.argtypes = [wintypes.HDC]
    user.PrintWindow.argtypes = [wintypes.HWND, wintypes.HDC, wintypes.UINT]
    dc = gdi.CreateCompatibleDC(None)
    header = struct.pack('<IiiHHIIiiII', 40, width, -height, 1, 32, 0, width * height * 4, 0, 0, 0, 0)
    info = ctypes.create_string_buffer(header)
    pixels = ctypes.c_void_p()
    bitmap = gdi.CreateDIBSection(dc, info, 0, ctypes.byref(pixels), None, 0)
    if not bitmap:
        gdi.DeleteDC(dc)
        raise ctypes.WinError(ctypes.get_last_error())
    previous = gdi.SelectObject(dc, bitmap)
    try:
        if not user.PrintWindow(window, dc, 2):
            raise RuntimeError('Could not capture updater window')
        data = ctypes.string_at(pixels, width * height * 4)
        output.write_bytes(struct.pack('<2sIHHI', b'BM', 54 + len(data), 0, 0, 54) + header + data)
    finally:
        gdi.SelectObject(dc, previous)
        gdi.DeleteObject(bitmap)
        gdi.DeleteDC(dc)
