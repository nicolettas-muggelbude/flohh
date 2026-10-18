// Flohh kopiert Bibliotheken und Ordner per LAN-Kabel von einem
// Rechner auf einen anderen.
//
// Aufruf:
//
//	flohh            startet die Oberfläche (wie "flohh gui")
//	flohh gui        startet die Oberfläche im Browser
//	flohh version    zeigt die Version
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nicolettas-muggelbude/flohh/internal/browser"
	"github.com/nicolettas-muggelbude/flohh/internal/ui"
)

// version wird beim Bauen gesetzt: go build -ldflags "-X main.version=1.2.3"
var version = "dev"

func main() {
	// main bleibt bewusst kurz: Die eigentliche Arbeit macht run, das einen
	// Fehler zurückgibt. So gibt es genau eine Stelle, die Fehler ausgibt.
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "Fehler:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	cmd := "gui"
	if len(args) > 0 {
		cmd, args = args[0], args[1:]
	}

	switch cmd {
	case "gui":
		return runGUI(args)
	case "version", "--version", "-v":
		fmt.Println("flohh", version)
		return nil
	case "help", "--help", "-h":
		printHelp()
		return nil
	default:
		printHelp()
		return fmt.Errorf("unbekannter Befehl %q", cmd)
	}
}

func printHelp() {
	fmt.Println(`Flohh – Daten springen von Rechner zu Rechner

Befehle:
  gui        Oberfläche im Browser starten (Standard)
  version    Version anzeigen
  help       diese Hilfe

Optionen für gui:
  -port N        fester Port für die Oberfläche (Standard: automatisch)
  -no-browser    Browser nicht automatisch öffnen`)
}

func runGUI(args []string) error {
	flags := flag.NewFlagSet("gui", flag.ContinueOnError)
	port := flags.Int("port", 0, "Port für die Oberfläche (0 = automatisch)")
	noBrowser := flags.Bool("no-browser", false, "Browser nicht automatisch öffnen")
	if err := flags.Parse(args); err != nil {
		return err
	}

	// Nur auf 127.0.0.1 lauschen: Die Oberfläche ist ausschließlich auf
	// diesem Rechner erreichbar, nie aus dem Netzwerk.
	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", *port))
	if err != nil {
		return fmt.Errorf("Oberfläche kann nicht starten: %w", err)
	}
	addr := ln.Addr().String() // z. B. "127.0.0.1:53817"
	url := "http://" + addr + "/"

	// ctx wird beendet, wenn Strg+C gedrückt wird, das System die App
	// beendet oder jemand in der Oberfläche auf "Beenden" klickt (stop).
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	server := &http.Server{
		Handler:           ui.New(version, addr, stop),
		ReadHeaderTimeout: 10 * time.Second,
	}

	// Der Server läuft in einer eigenen Goroutine (nebenläufig), damit main
	// weiterarbeiten und auf das Ende warten kann. Sein Ergebnis kommt über
	// den Channel serveErr zurück.
	serveErr := make(chan error, 1)
	go func() { serveErr <- server.Serve(ln) }()

	log.Printf("Flohh %s läuft auf %s", version, url)
	if !*noBrowser {
		if err := browser.Open(url); err != nil {
			log.Printf("Browser ließ sich nicht öffnen (%v). Bitte %s selbst öffnen.", err, url)
		}
	}

	// Warten, bis entweder beendet werden soll oder der Server abbricht.
	select {
	case <-ctx.Done():
	case err := <-serveErr:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	}

	log.Println("Flohh wird beendet …")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return server.Shutdown(shutdownCtx)
}
