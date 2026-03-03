package main

import (
	"fmt"
	"os"

	"atlas.grave/internal/system"
	"atlas.grave/internal/ui"
	tea "github.com/charmbracelet/bubbletea"
)

var Version = "dev"

func main() {
	if len(os.Args) > 1 && (os.Args[1] == "-v" || os.Args[1] == "--version") {
		fmt.Printf("atlas.grave v%s\n", Version)
		return
	}
	if len(os.Args) > 1 && (os.Args[1] == "-h" || os.Args[1] == "--help" || os.Args[1] == "help") {
		fmt.Println("Atlas Grave - High-fidelity interactive process reaper.")
		fmt.Println("\nUsage:")
		fmt.Println("  atlas.grave        Start the process reaper TUI")
		fmt.Println("  atlas.grave -v     Show version")
		fmt.Println("  atlas.grave -h     Show this help")
		return
	}

	reaper := system.NewReaper()
	p := tea.NewProgram(ui.NewModel(reaper), tea.WithAltScreen())
	
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
