package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"text/template"

	"github.com/spf13/cobra"
)

const launchdPlist = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.hrishikeshs.sift</string>
    <key>ProgramArguments</key>
    <array>
        <string>{{.Binary}}</string>
        <string>daemon</string>
        <string>--interval</string>
        <string>60m</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>{{.LogDir}}/sift.log</string>
    <key>StandardErrorPath</key>
    <string>{{.LogDir}}/sift.err</string>
</dict>
</plist>
`

const systemdUnit = `[Unit]
Description=sift - Claude Code session indexer
After=default.target

[Service]
Type=simple
ExecStart={{.Binary}} daemon --interval 60m
Restart=on-failure
RestartSec=30

[Install]
WantedBy=default.target
`

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Set up automatic indexing as a background service",
	RunE: func(cmd *cobra.Command, args []string) error {
		binary, err := exec.LookPath("sift")
		if err != nil {
			binary, _ = os.Executable()
		}

		switch runtime.GOOS {
		case "darwin":
			return installLaunchd(binary)
		case "linux":
			return installSystemd(binary)
		default:
			return fmt.Errorf("unsupported platform: %s (use 'sift daemon' manually)", runtime.GOOS)
		}
	},
}

var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Remove the automatic indexing service",
	RunE: func(cmd *cobra.Command, args []string) error {
		switch runtime.GOOS {
		case "darwin":
			return uninstallLaunchd()
		case "linux":
			return uninstallSystemd()
		default:
			return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
		}
	},
}

func init() {
	rootCmd.AddCommand(installCmd)
	rootCmd.AddCommand(uninstallCmd)
}

func installLaunchd(binary string) error {
	home, _ := os.UserHomeDir()
	plistPath := filepath.Join(home, "Library", "LaunchAgents", "com.hrishikeshs.sift.plist")
	logDir := filepath.Join(home, ".sift", "logs")
	os.MkdirAll(logDir, 0755)

	tmpl, err := template.New("plist").Parse(launchdPlist)
	if err != nil {
		return err
	}

	f, err := os.Create(plistPath)
	if err != nil {
		return fmt.Errorf("creating plist: %w", err)
	}
	defer f.Close()

	if err := tmpl.Execute(f, map[string]string{"Binary": binary, "LogDir": logDir}); err != nil {
		return err
	}

	out, err := exec.Command("launchctl", "load", plistPath).CombinedOutput()
	if err != nil {
		return fmt.Errorf("loading service: %s: %w", string(out), err)
	}

	fmt.Printf("Installed and started sift daemon\n")
	fmt.Printf("  Plist: %s\n", plistPath)
	fmt.Printf("  Logs:  %s/\n", logDir)
	return nil
}

func uninstallLaunchd() error {
	home, _ := os.UserHomeDir()
	plistPath := filepath.Join(home, "Library", "LaunchAgents", "com.hrishikeshs.sift.plist")

	exec.Command("launchctl", "unload", plistPath).Run()
	if err := os.Remove(plistPath); err != nil && !os.IsNotExist(err) {
		return err
	}

	fmt.Println("Uninstalled sift daemon")
	return nil
}

func installSystemd(binary string) error {
	home, _ := os.UserHomeDir()
	unitDir := filepath.Join(home, ".config", "systemd", "user")
	unitPath := filepath.Join(unitDir, "sift.service")
	os.MkdirAll(unitDir, 0755)

	tmpl, err := template.New("unit").Parse(systemdUnit)
	if err != nil {
		return err
	}

	f, err := os.Create(unitPath)
	if err != nil {
		return fmt.Errorf("creating unit file: %w", err)
	}
	defer f.Close()

	if err := tmpl.Execute(f, map[string]string{"Binary": binary}); err != nil {
		return err
	}

	exec.Command("systemctl", "--user", "daemon-reload").Run()
	out, err := exec.Command("systemctl", "--user", "enable", "--now", "sift.service").CombinedOutput()
	if err != nil {
		return fmt.Errorf("enabling service: %s: %w", string(out), err)
	}

	fmt.Printf("Installed and started sift daemon\n")
	fmt.Printf("  Unit: %s\n", unitPath)
	return nil
}

func uninstallSystemd() error {
	exec.Command("systemctl", "--user", "disable", "--now", "sift.service").Run()

	home, _ := os.UserHomeDir()
	unitPath := filepath.Join(home, ".config", "systemd", "user", "sift.service")
	if err := os.Remove(unitPath); err != nil && !os.IsNotExist(err) {
		return err
	}

	exec.Command("systemctl", "--user", "daemon-reload").Run()
	fmt.Println("Uninstalled sift daemon")
	return nil
}
