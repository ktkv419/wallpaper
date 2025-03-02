//go:build windows
// +build windows

package wallpaper

import (
	"log"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"unicode/utf16"
	"unsafe"

	"golang.org/x/sys/windows/registry"
)

// https://msdn.microsoft.com/en-us/library/windows/desktop/ms724947.aspx
const (
	spiGetDeskWallpaper = 0x0073
	spiSetDeskWallpaper = 0x0014

	uiParam = 0x0000

	spifUpdateINIFile = 0x01
	spifSendChange    = 0x02
)

// https://msdn.microsoft.com/en-us/library/windows/desktop/ms724947.aspx
var (
	user32               = syscall.NewLazyDLL("user32.dll")
	systemParametersInfo = user32.NewProc("SystemParametersInfoW")
)

// Checks if the script is running as Administrator
func isAdmin() bool {
	cmd := exec.Command("net", "session")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	err := cmd.Run()
	return err == nil
}

// Relaunches the script with Administrator privileges
func runAsAdmin() {
	exe, err := os.Executable()
	if err != nil {
		log.Fatalf("Failed to get executable path: %v", err)
	}

	cmd := exec.Command("powershell", "Start-Process", exe, "-Verb", "RunAs")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	err = cmd.Run()
	if err != nil {
		log.Fatalf("Failed to run as administrator: %v", err)
	}

	os.Exit(0) // Exit current process, new one will start with admin rights
}

// Get returns the current wallpaper.
func Get() (string, error) {
	// the maximum length of a windows path is 256 utf16 characters
	var filename [256]uint16
	systemParametersInfo.Call(
		uintptr(spiGetDeskWallpaper),
		uintptr(cap(filename)),
		// the memory address of the first byte of the array
		uintptr(unsafe.Pointer(&filename[0])),
		uintptr(0),
	)
	return strings.Trim(string(utf16.Decode(filename[:])), "\x00"), nil
}

func checkRegistryValues(filename string) bool {
	expectedValues := map[string]interface{}{
		`SOFTWARE\Policies\Microsoft\Windows\Personalization`: map[string]interface{}{
			"LockScreenImage": filename,
		},
		`SOFTWARE\Microsoft\Windows\CurrentVersion\PersonalizationCSP`: map[string]interface{}{
			"LockScreenImageStatus": uint32(1),
			"LockScreenImagePath":   filename,
			"LockScreenImageUrl":    filename,
		},
	}

	for keyPath, values := range expectedValues {
		// Open registry key
		key, err := registry.OpenKey(registry.LOCAL_MACHINE, keyPath, registry.QUERY_VALUE)
		if err != nil {
			return false
		}
		defer key.Close()

		// Iterate through expected values
		for valueName, expected := range values.(map[string]interface{}) {
			switch expected := expected.(type) {
			case string:
				val, _, err := key.GetStringValue(valueName)
				if err != nil || val != expected {
					return false
				}
			case uint32:
				val, _, err := key.GetIntegerValue(valueName)
				if err != nil || val != uint64(expected) {
					return false
				}
			}
		}
	}

	return true
}

func setLockscreen(filename string) error {
	if !isAdmin() {
		runAsAdmin()
		return nil // Exit after requesting elevation
	}

	// Set lockscreen
	key, _, err := registry.CreateKey(registry.LOCAL_MACHINE, `SOFTWARE\Policies\Microsoft\Windows\Personalization`, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer key.Close()
	if err := key.SetStringValue("LockScreenImage", filename); err != nil {
		return err
	}

	// Set PersonalizationCSP settings
	cspKey, _, err := registry.CreateKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows\CurrentVersion\PersonalizationCSP`, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer cspKey.Close()

	if err := cspKey.SetDWordValue("LockScreenImageStatus", 1); err != nil {
		return err
	}
	if err := cspKey.SetStringValue("LockScreenImagePath", filename); err != nil {
		return err
	}
	if err := cspKey.SetStringValue("LockScreenImageUrl", filename); err != nil {
		return err
	}
	return nil
}

// SetFromFile sets the wallpaper for the current user.
func SetFromFile(filename string) error {
	filenameUTF16, err := syscall.UTF16PtrFromString(filename)
	if err != nil {
		return err
	}

	if !checkRegistryValues(filename) {
		err := setLockscreen(filename)

		if err != nil {
			return err
		}
	}

	systemParametersInfo.Call(
		uintptr(spiSetDeskWallpaper),
		uintptr(uiParam),
		uintptr(unsafe.Pointer(filenameUTF16)),
		uintptr(spifUpdateINIFile|spifSendChange),
	)

	return nil
}

// SetMode sets the wallpaper mode.
func SetMode(mode Mode) error {
	key, _, err := registry.CreateKey(registry.CURRENT_USER, "Control Panel\\Desktop", registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer key.Close()

	var tile string
	if mode == Tile {
		tile = "1"
	} else {
		tile = "0"
	}
	err = key.SetStringValue("TileWallpaper", tile)
	if err != nil {
		return err
	}

	var style string
	switch mode {
	case Center, Tile:
		style = "0"
	case Fit:
		style = "6"
	case Span:
		style = "22"
	case Stretch:
		style = "2"
	case Crop:
		style = "10"
	default:
		panic("invalid wallpaper mode")
	}
	err = key.SetStringValue("WallpaperStyle", style)
	if err != nil {
		return err
	}

	// updates wallpaper
	path, err := Get()
	if err != nil {
		return err
	}

	return SetFromFile(path)
}

func getCacheDir() (string, error) {
	return os.TempDir(), nil
}
