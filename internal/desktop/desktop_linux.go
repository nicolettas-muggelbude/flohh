package desktop

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// paths liefert, wohin flohh, Icon und Menü-Eintrag kommen
// (nach freedesktop.org-Standard, nur im eigenen Benutzerordner).
type paths struct{ bin, icon, entry string }

func userPaths() (paths, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return paths{}, err
	}
	data := os.Getenv("XDG_DATA_HOME")
	if data == "" {
		data = filepath.Join(home, ".local", "share")
	}
	return paths{
		bin:   filepath.Join(home, ".local", "bin", "flohh"),
		icon:  filepath.Join(data, "icons", "hicolor", "scalable", "apps", "flohh.svg"),
		entry: filepath.Join(data, "applications", "flohh.desktop"),
	}, nil
}

func running() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	if r, err := filepath.EvalSymlinks(exe); err == nil {
		exe = r
	}
	return exe
}

// State meldet, ob die Aufnahme ins Menü angeboten werden soll: nicht, wenn
// flohh schon aus dem Menü-Ordner läuft oder jemand „Nein danke“ gesagt hat.
func State() Status {
	p, err := userPaths()
	if err != nil {
		return Status{}
	}
	_, statErr := os.Stat(p.entry)
	installed := statErr == nil
	return Status{
		Installed: installed,
		Offer:     running() != p.bin && !loadSettings().MenuDeclined,
	}
}

// Install kopiert die laufende flohh-Datei nach ~/.local/bin und legt Icon
// und Menü-Eintrag an. Eine ältere Fassung wird dabei ersetzt.
func Install(icon []byte) error {
	p, err := userPaths()
	if err != nil {
		return err
	}
	for _, f := range []string{p.bin, p.icon, p.entry} {
		if err := os.MkdirAll(filepath.Dir(f), 0o755); err != nil {
			return err
		}
	}
	if exe := running(); exe != p.bin {
		if err := copyExecutable(exe, p.bin); err != nil {
			return err
		}
	}
	if err := os.WriteFile(p.icon, icon, 0o644); err != nil {
		return err
	}
	entry := fmt.Sprintf(`[Desktop Entry]
Type=Application
Name=flohh
GenericName=PC-Umzug
Comment=Daten springen von Rechner zu Rechner
Exec="%s"
Icon=%s
Terminal=false
Categories=Utility;Network;FileTransfer;
Keywords=Umzug;Übertragung;Kopieren;Transfer;
`, p.bin, p.icon)
	return os.WriteFile(p.entry, []byte(entry), 0o644)
}

// copyExecutable schreibt erst eine Hilfsdatei und benennt sie dann um. So
// klappt das Ersetzen auch, wenn die alte Fassung gerade läuft.
func copyExecutable(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	tmp := dst + ".neu"
	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		os.Remove(tmp)
		return err
	}
	if err := out.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, dst)
}
