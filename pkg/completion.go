package pkg

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	_ "syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

func DetectShell() string {
	// Unix case
	if sh := os.Getenv("SHELL"); sh != "" {
		return filepath.Base(sh)
	}

	// Windows case
	if runtime.GOOS == "windows" {
		exe, err := getParentProcessPath()
		if err != nil {
			return "powershell" // fallback
		}

		exe = strings.ToLower(exe)

		switch {
		case strings.Contains(exe, "powershell"):
			return "powershell"
		case strings.Contains(exe, "pwsh"):
			return "powershell"
		case strings.Contains(exe, "cmd.exe"):
			return "cmd"
		case strings.Contains(exe, "bash.exe"):
			return "bash" // Git Bash
		}
		return "powershell"
	}

	return "unknown"
}

func getParentProcessPath() (string, error) {
	ppid := uint32(os.Getppid())

	// Open process handle
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_INFORMATION|windows.PROCESS_VM_READ, false, ppid)
	if err != nil {
		return "", err
	}
	defer windows.CloseHandle(h)

	// Resolve Psapi.GetModuleFileNameExW
	modPsapi := windows.NewLazySystemDLL("psapi.dll")
	procGetModuleFileNameExW := modPsapi.NewProc("GetModuleFileNameExW")

	buf := make([]uint16, windows.MAX_PATH)

	r0, _, e1 := procGetModuleFileNameExW.Call(
		uintptr(h),
		0,
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(len(buf)),
	)

	if r0 == 0 {
		return "", e1
	}

	return windows.UTF16ToString(buf), nil
}
