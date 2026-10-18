package peer

import (
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"time"

	"filippo.io/cpace"
)

// Absicherung einer frischen TCP-Verbindung in zwei Stufen:
//
//  1. TLS 1.3 mit Wegwerf-Zertifikaten. Ab hier ist alles verschlüsselt,
//     aber noch nicht klar, wer am anderen Ende sitzt.
//  2. CPace, ein PAKE-Verfahren, mit dem 6-stelligen Code. Es ist an genau
//     diese TLS-Verbindung gebunden (TLS-Exporter). Ein Angreifer dazwischen
//     kann pro Verbindung nur einen einzigen Code raten; ein einfacher
//     Code-Vergleich ließe sich dagegen in Millisekunden durchprobieren.
//
// Danach beweisen beide per HMAC, dass sie denselben Schlüssel haben.
// Die TLS-Client-Seite startet CPace, die Server-Seite antwortet.

// errWrongCode: Die Gegenseite hat einen anderen Code benutzt.
var errWrongCode = errors.New("wrong_code")

const (
	maxFrame      = 4096
	exporterLabel = "EXPORTER-flohh-pairing"
)

// hello schickt die anrufende Seite direkt nach dem TLS-Handshake. Erst nach
// erfolgreicher Kopplung ist sicher, dass die Angaben stimmen.
type hello struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	OS   string `json:"os"`
	Role Role   `json:"role"`
}

func newTLSConfig() (*tls.Config, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "flohh"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(7 * 24 * time.Hour),
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, pub, priv)
	if err != nil {
		return nil, err
	}
	return &tls.Config{
		Certificates: []tls.Certificate{{Certificate: [][]byte{der}, PrivateKey: priv}},
		MinVersion:   tls.VersionTLS13,
		// Kein Zertifikat wird geprüft: Wer am anderen Ende sitzt, beweist die
		// Kopplung per Code, die an genau diese Verbindung gebunden ist.
		InsecureSkipVerify: true,
	}, nil
}

// newCode liefert einen zufälligen 6-stelligen Code.
func newCode() string {
	n, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		panic(err) // crypto/rand fällt nicht aus
	}
	return fmt.Sprintf("%06d", n.Int64())
}

func writeFrame(w io.Writer, p []byte) error {
	if len(p) > maxFrame {
		return errors.New("Nachricht zu groß")
	}
	var h [2]byte
	binary.BigEndian.PutUint16(h[:], uint16(len(p)))
	if _, err := w.Write(h[:]); err != nil {
		return err
	}
	_, err := w.Write(p)
	return err
}

func readFrame(r io.Reader) ([]byte, error) {
	var h [2]byte
	if _, err := io.ReadFull(r, h[:]); err != nil {
		return nil, err
	}
	n := binary.BigEndian.Uint16(h[:])
	if n > maxFrame {
		return nil, errors.New("Nachricht zu groß")
	}
	p := make([]byte, n)
	_, err := io.ReadFull(r, p)
	return p, err
}

func writeHello(c *tls.Conn, h hello) error {
	data, err := json.Marshal(h)
	if err != nil {
		return err
	}
	return writeFrame(c, data)
}

func readHello(c *tls.Conn) (hello, error) {
	var h hello
	data, err := readFrame(c)
	if err != nil {
		return h, err
	}
	err = json.Unmarshal(data, &h)
	return h, err
}

func contextInfo(c *tls.Conn) (*cpace.ContextInfo, error) {
	cs := c.ConnectionState()
	ekm, err := cs.ExportKeyingMaterial(exporterLabel, nil, 32)
	if err != nil {
		return nil, err
	}
	return cpace.NewContextInfo("flohh-tls-client", "flohh-tls-server", ekm), nil
}

func confirmTag(key []byte, side string) []byte {
	m := hmac.New(sha256.New, key)
	m.Write([]byte("flohh confirm " + side))
	return m.Sum(nil)
}

// pakeClient koppelt als TLS-Client. Blockiert, bis die Gegenseite ihren
// Code kennt (beim Sender also, bis jemand ihn eingetippt hat).
func pakeClient(c *tls.Conn, code string) error {
	ci, err := contextInfo(c)
	if err != nil {
		return err
	}
	msgA, st, err := cpace.Start(code, ci)
	if err != nil {
		return err
	}
	if err := writeFrame(c, msgA); err != nil {
		return err
	}
	msgB, err := readFrame(c)
	if err != nil {
		return err
	}
	key, err := st.Finish(msgB)
	if err != nil {
		return err
	}
	tag, err := readFrame(c)
	if err != nil {
		return err
	}
	if !hmac.Equal(tag, confirmTag(key, "server")) {
		return errWrongCode
	}
	return writeFrame(c, confirmTag(key, "client"))
}

// pakeServer koppelt als TLS-Server.
func pakeServer(c *tls.Conn, code string) error {
	ci, err := contextInfo(c)
	if err != nil {
		return err
	}
	msgA, err := readFrame(c)
	if err != nil {
		return err
	}
	msgB, key, err := cpace.Exchange(code, ci, msgA)
	if err != nil {
		return err
	}
	if err := writeFrame(c, msgB); err != nil {
		return err
	}
	if err := writeFrame(c, confirmTag(key, "server")); err != nil {
		return err
	}
	tag, err := readFrame(c)
	if err != nil {
		// Der Client legt auf, wenn unser Nachweis nicht passt – also bei
		// falschem Code.
		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
			return errWrongCode
		}
		return err
	}
	if !hmac.Equal(tag, confirmTag(key, "client")) {
		return errWrongCode
	}
	return nil
}
