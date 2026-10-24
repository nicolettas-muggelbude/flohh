package transfer

import (
	"fmt"
	"io"
	"sync"
)

// RemoteError meldet der Sender (z. B. Datei nicht lesbar). Die Verbindung
// selbst ist dann in Ordnung.
type RemoteError string

func (e RemoteError) Error() string { return "Sender: " + string(e) }

// localError: Schreiben auf diesem Rechner ist fehlgeschlagen (z. B. Platte
// voll). Die Verbindung ist in Ordnung.
type localError struct{ err error }

func (e localError) Error() string { return e.err.Error() }
func (e localError) Unwrap() error { return e.err }

// Client läuft auf dem neuen Rechner und stellt die Anfragen. Anfragen
// laufen nacheinander; der Client ist trotzdem sicher für mehrere Goroutinen.
type Client struct {
	mu sync.Mutex
	w  *wire
}

func NewClient(rw io.ReadWriter) *Client { return &Client{w: newWire(rw)} }

func (c *Client) send(req request) error {
	if err := c.w.writeJSON(req); err != nil {
		return err
	}
	return c.w.flush()
}

// Roots fragt die freigegebenen Ordner ab.
func (c *Client) Roots() ([]Root, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.send(request{Op: "roots"}); err != nil {
		return nil, err
	}
	var resp response
	if err := c.w.readJSON(&resp); err != nil {
		return nil, err
	}
	if resp.Error != "" {
		return nil, RemoteError(resp.Error)
	}
	return resp.Roots, nil
}

// Walk listet alle Dateien einer Freigabe.
func (c *Client) Walk(root string, fn func(Entry)) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.send(request{Op: "walk", Root: root}); err != nil {
		return err
	}
	for {
		typ, p, err := c.w.readFrame()
		if err != nil {
			return err
		}
		switch typ {
		case frameEnd:
			return nil
		case frameJSON:
			var resp response
			if err := jsonUnmarshal(p, &resp); err != nil {
				return err
			}
			if resp.Error != "" {
				return RemoteError(resp.Error)
			}
			for _, e := range resp.Entries {
				fn(e)
			}
		default:
			return fmt.Errorf("unerwartete Rahmenart %q", typ)
		}
	}
}

// Read holt eine Datei ab Offset und schreibt sie nach dst. progress geht
// zur Anzeige an den Sender. Schlägt das Schreiben fehl, wird der Rest der
// Datei trotzdem gelesen, damit die Verbindung im Takt bleibt.
func (c *Client) Read(root, path string, off int64, progress *Progress, dst io.Writer) (int64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.send(request{Op: "read", Root: root, Path: path, Offset: off, Progress: progress}); err != nil {
		return 0, err
	}
	var head response
	if err := c.w.readJSON(&head); err != nil {
		return 0, err
	}
	if head.Error != "" {
		return 0, RemoteError(head.Error)
	}
	var n int64
	var writeErr error
	for {
		typ, p, err := c.w.readFrame()
		if err != nil {
			return n, err
		}
		switch typ {
		case frameData:
			if writeErr == nil {
				if _, err := dst.Write(p); err != nil {
					writeErr = localError{err}
				}
			}
			n += int64(len(p))
		case frameEnd:
			if writeErr != nil {
				return n, writeErr
			}
			if n != head.Size {
				return n, RemoteError("file_changed")
			}
			return n, nil
		case frameJSON:
			var resp response
			jsonUnmarshal(p, &resp)
			return n, RemoteError(resp.Error)
		default:
			return n, fmt.Errorf("unerwartete Rahmenart %q", typ)
		}
	}
}

// Done meldet dem Sender das Ergebnis.
func (c *Client) Done(p Progress) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.send(request{Op: "done", Progress: &p}); err != nil {
		return err
	}
	var resp response
	return c.w.readJSON(&resp)
}
