// Package transfer überträgt Ordner über eine gekoppelte Verbindung.
//
// Der neue Rechner (Empfänger) holt ab, der alte (Sender) liefert nur und
// nur aus Ordnern, die er freigegeben hat. Das Protokoll besteht aus
// einzelnen Anfragen (roots, walk, read, done) – dieselben Bausteine kann
// später ein Dateimanager benutzen.
package transfer

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// Rahmenarten auf der Leitung: [Art 1 Byte][Länge 4 Byte][Inhalt].
const (
	frameJSON byte = 'J' // Anfrage oder Antwort als JSON
	frameData byte = 'D' // Dateiinhalt
	frameEnd  byte = 'E' // Ende einer mehrteiligen Antwort
)

const (
	maxFrame  = 1 << 20  // 1 MiB
	chunkSize = 64 << 10 // Dateiinhalt in Stücken zu 64 KiB
)

type wire struct {
	r *bufio.Reader
	w *bufio.Writer
}

func newWire(rw io.ReadWriter) *wire {
	return &wire{r: bufio.NewReaderSize(rw, 128<<10), w: bufio.NewWriterSize(rw, 128<<10)}
}

func (w *wire) writeFrame(typ byte, p []byte) error {
	if len(p) > maxFrame {
		return errors.New("Rahmen zu groß")
	}
	var h [5]byte
	h[0] = typ
	binary.BigEndian.PutUint32(h[1:], uint32(len(p)))
	if _, err := w.w.Write(h[:]); err != nil {
		return err
	}
	_, err := w.w.Write(p)
	return err
}

func (w *wire) writeJSON(v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return w.writeFrame(frameJSON, data)
}

func (w *wire) flush() error { return w.w.Flush() }

func (w *wire) readFrame() (byte, []byte, error) {
	var h [5]byte
	if _, err := io.ReadFull(w.r, h[:]); err != nil {
		return 0, nil, err
	}
	n := binary.BigEndian.Uint32(h[1:])
	if n > maxFrame {
		return 0, nil, errors.New("Rahmen zu groß")
	}
	p := make([]byte, n)
	_, err := io.ReadFull(w.r, p)
	return h[0], p, err
}

func (w *wire) readJSON(v any) error {
	typ, p, err := w.readFrame()
	if err != nil {
		return err
	}
	if typ != frameJSON {
		return fmt.Errorf("JSON erwartet, Rahmenart %q bekommen", typ)
	}
	return json.Unmarshal(p, v)
}

// request schickt der Empfänger an den Sender.
type request struct {
	Op       string    `json:"op"` // roots | walk | read | done
	Root     string    `json:"root,omitempty"`
	Path     string    `json:"path,omitempty"` // relativ, Trenner „/“
	Offset   int64     `json:"offset,omitempty"`
	Progress *Progress `json:"progress,omitempty"` // Stand für die Anzeige beim Sender
}

// response ist die Antwort des Senders (bei walk mehrere, bei read gefolgt
// von Datenrahmen).
type response struct {
	Error   string  `json:"error,omitempty"`
	By      string  `json:"by,omitempty"` // bei „locked“: Programm, das die Datei geöffnet hat
	Roots   []Root  `json:"roots,omitempty"`
	Entries []Entry `json:"entries,omitempty"`
	Size    int64   `json:"size,omitempty"`
}

// Root ist ein freigegebener Ordner. ID ist die Art (folders.Kind) oder bei
// eigenen Ordnern „custom-N“.
type Root struct {
	ID     string `json:"id"`
	Name   string `json:"name,omitempty"`   // Ordnername (für eigene Ordner)
	Source string `json:"source,omitempty"` // Pfad auf dem alten Rechner (zur Anzeige)
	Stats         // Dateien, Bytes, bewusst Weggelassenes
	Ready  bool   `json:"ready"` // Zählung fertig
}

// Entry ist eine Datei in einem freigegebenen Ordner.
type Entry struct {
	Path  string `json:"path"` // relativ zur Freigabe, Trenner „/“
	Size  int64  `json:"size"`
	MTime int64  `json:"mtime"` // Unix-Zeit in Sekunden
}
