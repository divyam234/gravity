package logger

import (
	"fmt"
	"strings"
)

const banner = `
   ____                 _ _         
  / ___|_ __ __ ___   _(_) |_ _   _ 
 | |  _| '__/ _` + "`" + ` \ \ / / | __| | | |
 | |_| | | | (_| |\ V /| | |_| |_| |
  \____|_|  \__,_| \_/ |_|\__|\__, |
                              |___/
`

type StartupInfo struct {
	Version  string
	Addr     string
	DataDir  string
	LogLevel string
}

func PrintBanner(info StartupInfo) {
	fmt.Print(banner)
	fmt.Printf("                              v%s\n", info.Version)
	fmt.Println()

	maxWidth := 50
	fmt.Printf("  %s\n", strings.Repeat("─", maxWidth))
	fmt.Printf("  → Address:  http://%s\n", formatAddr(info.Addr))
	fmt.Printf("  → Data Dir: %s\n", info.DataDir)
	fmt.Printf("  → Log Level: %s\n", info.LogLevel)
	fmt.Printf("  %s\n", strings.Repeat("─", maxWidth))
	fmt.Println()
}

func formatAddr(addr string) string {
	if strings.HasPrefix(addr, ":") {
		return "localhost" + addr
	}
	return addr
}
