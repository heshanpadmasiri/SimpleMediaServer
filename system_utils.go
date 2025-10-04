package main

import (
	"fmt"
	"os/exec"
	"runtime"
)

// SystemUtils interface for system operations
type SystemUtils interface {
	MoveToTrash(path string) error
}

// LinuxSystemUtils implements SystemUtils for Linux systems
type LinuxSystemUtils struct{}

func (l LinuxSystemUtils) MoveToTrash(path string) error {
	// Prefer gio (GLib) trash which adheres to the FreeDesktop Trash spec
	if err := tryExec("gio", "trash", path); err == nil {
		return nil
	}
	// Older gvfs-trash
	if err := tryExec("gvfs-trash", path); err == nil {
		return nil
	}
	// trash-put from trash-cli
	if err := tryExec("trash-put", path); err == nil {
		return nil
	}
	// KDE kioclient5
	if err := tryExec("kioclient5", "move", path, "trash:/"); err == nil {
		return nil
	}
	return fmt.Errorf("no trash utility found (tried gio, gvfs-trash, trash-put, kioclient5)")
}

// WindowsSystemUtils implements SystemUtils for Windows systems
type WindowsSystemUtils struct{}

func (w WindowsSystemUtils) MoveToTrash(path string) error {
	// Use PowerShell to move file to Recycle Bin
	cmd := exec.Command("powershell", "-Command",
		fmt.Sprintf("Add-Type -AssemblyName Microsoft.VisualBasic; [Microsoft.VisualBasic.FileIO.FileSystem]::DeleteFile('%s', 'OnlyRecycleBin')", path))
	return cmd.Run()
}

// MacOSSystemUtils implements SystemUtils for macOS systems
type MacOSSystemUtils struct{}

func (m MacOSSystemUtils) MoveToTrash(path string) error {
	// Use osascript to move file to Trash
	cmd := exec.Command("osascript", "-e", fmt.Sprintf("tell application \"Finder\" to move POSIX file \"%s\" to trash", path))
	return cmd.Run()
}

// NewSystemUtils creates the appropriate SystemUtils implementation based on the operating system
func NewSystemUtils() SystemUtils {
	switch runtime.GOOS {
	case "windows":
		return WindowsSystemUtils{}
	case "darwin":
		return MacOSSystemUtils{}
	case "linux":
		return LinuxSystemUtils{}
	default:
		// Default to Linux for other Unix-like systems
		return LinuxSystemUtils{}
	}
}

// tryExec is a helper function to try executing a command
func tryExec(name string, args ...string) error {
	if _, err := exec.LookPath(name); err != nil {
		return err
	}
	cmd := exec.Command(name, args...)
	return cmd.Run()
}
