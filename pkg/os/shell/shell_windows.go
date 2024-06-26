package shell

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"
)

var (
	supportedShell = []string{"cmd", "powershell"}
)

// re-implementation of private function in https://github.com/golang/go/blob/master/src/syscall/syscall_windows.go
func getProcessEntry(pid int) (pe *syscall.ProcessEntry32, err error) {
	snapshot, err := syscall.CreateToolhelp32Snapshot(syscall.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = syscall.CloseHandle(syscall.Handle(snapshot))
	}()

	var processEntry syscall.ProcessEntry32
	processEntry.Size = uint32(unsafe.Sizeof(processEntry))
	err = syscall.Process32First(snapshot, &processEntry)
	if err != nil {
		return nil, err
	}

	for {
		if processEntry.ProcessID == uint32(pid) {
			pe = &processEntry
			return
		}

		err = syscall.Process32Next(snapshot, &processEntry)
		if err != nil {
			return nil, err
		}
	}
}

// getNameAndItsPpid returns the exe file name its parent process id.
func getNameAndItsPpid(pid int) (exefile string, parentid int, err error) {
	pe, err := getProcessEntry(pid)
	if err != nil {
		return "", 0, err
	}

	name := syscall.UTF16ToString(pe.ExeFile[:])
	return name, int(pe.ParentProcessID), nil
}

func detect() (string, error) {
	shell := os.Getenv("SHELL")

	if shell == "" {
		shell, shellppid, err := getNameAndItsPpid(os.Getppid())
		if err != nil {
			return "cmd", err // defaulting to cmd
		}
		switch {
		case strings.Contains(strings.ToLower(shell), "powershell"):
			return "powershell", nil
		case strings.Contains(strings.ToLower(shell), "pwsh"):
			return "powershell", nil
		case strings.Contains(strings.ToLower(shell), "cmd"):
			return "cmd", nil
		default:
			shell, _, err := getNameAndItsPpid(shellppid)
			if err != nil {
				return "cmd", err // defaulting to cmd
			}
			switch {
			case strings.Contains(strings.ToLower(shell), "powershell"):
				return "powershell", nil
			case strings.Contains(strings.ToLower(shell), "cmd"):
				return "cmd", nil
			default:
				return "cmd", nil // this could be either powershell or cmd, defaulting to cmd
			}
		}
	}

	if os.Getenv("__fish_bin_dir") != "" {
		return "fish", nil
	}

	return filepath.Base(shell), nil
}
