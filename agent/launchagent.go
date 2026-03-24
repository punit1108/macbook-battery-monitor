package agent

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"
)

const plistLabel = "com.ppunit.volt"
const plistTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>{{.Label}}</string>
    <key>ProgramArguments</key>
    <array>
        <string>{{.BinaryPath}}</string>
        <string>daemon</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <dict>
        <key>Crashed</key>
        <true/>
    </dict>
    <key>StandardOutPath</key>
    <string>{{.LogPath}}</string>
    <key>StandardErrorPath</key>
    <string>{{.LogPath}}</string>
</dict>
</plist>
`

type plistData struct {
	Label      string
	BinaryPath string
	LogPath    string
}

// Install copies the current binary to ~/.local/bin and installs the LaunchAgent.
func Install() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	// Resolve paths.
	binDir := filepath.Join(home, ".local", "bin")
	binPath := filepath.Join(binDir, "volt")
	logDir := filepath.Join(home, ".volt")
	logPath := filepath.Join(logDir, "daemon.log")
	agentsDir := filepath.Join(home, "Library", "LaunchAgents")
	plistPath := filepath.Join(agentsDir, plistLabel+".plist")

	// Create directories.
	for _, dir := range []string{binDir, logDir, agentsDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("mkdir %s: %w", dir, err)
		}
	}

	// Copy current executable.
	self, err := os.Executable()
	if err != nil {
		return fmt.Errorf("find executable: %w", err)
	}
	if err := copyFile(self, binPath); err != nil {
		return fmt.Errorf("copy binary: %w", err)
	}
	if err := os.Chmod(binPath, 0755); err != nil {
		return err
	}
	fmt.Println("✓ binary installed to", binPath)

	// Write plist.
	tmpl := template.Must(template.New("plist").Parse(plistTemplate))
	f, err := os.Create(plistPath)
	if err != nil {
		return fmt.Errorf("create plist: %w", err)
	}
	defer f.Close()
	if err := tmpl.Execute(f, plistData{
		Label:      plistLabel,
		BinaryPath: binPath,
		LogPath:    logPath,
	}); err != nil {
		return err
	}
	fmt.Println("✓ LaunchAgent plist written to", plistPath)

	// Load via launchctl.
	out, err := exec.Command("launchctl", "load", plistPath).CombinedOutput()
	if err != nil {
		return fmt.Errorf("launchctl load: %w\n%s", err, out)
	}
	fmt.Println("✓ daemon started via launchctl")
	fmt.Println()
	fmt.Println("The daemon will auto-start on every login and restart if it crashes.")
	fmt.Println("Logs:", logPath)
	fmt.Println("Data:", filepath.Join(logDir, "data/"))
	fmt.Println()
	fmt.Println("To stop and remove: volt uninstall")
	return nil
}

// Uninstall unloads the LaunchAgent and removes the binary and plist.
func Uninstall() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	plistPath := filepath.Join(home, "Library", "LaunchAgents", plistLabel+".plist")
	binPath := filepath.Join(home, ".local", "bin", "volt")

	// Unload — ignore error if not loaded.
	out, err := exec.Command("launchctl", "unload", plistPath).CombinedOutput()
	if err != nil {
		fmt.Println("note: launchctl unload:", string(out))
	} else {
		fmt.Println("✓ daemon stopped")
	}

	// Remove plist.
	if err := os.Remove(plistPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove plist: %w", err)
	}
	fmt.Println("✓ plist removed")

	// Remove binary.
	if err := os.Remove(binPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove binary: %w", err)
	}
	fmt.Println("✓ binary removed")

	fmt.Println()
	fmt.Println("Uninstall complete. Your collected data is preserved at ~/.volt/data/")
	return nil
}

// copyFile copies src to dst.
func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0755)
}
