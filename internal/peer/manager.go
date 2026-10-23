package peer

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"errors"
	"net"
	"os"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Fehlercodes für die Oberfläche (die übersetzt sie, siehe web/i18n.js).
const (
	ErrBadCode       = "bad_code"       // kein 6-stelliger Code
	ErrWrongCode     = "wrong_code"     // Code passt nicht
	ErrPeerGone      = "peer_gone"      // Gegenstelle nicht mehr da
	ErrConnectFailed = "connect_failed" // Verbindung kam nicht zustande
	ErrWrongRole     = "wrong_role"     // Aktion passt nicht zur Rolle
	ErrNetwork       = "network"        // Ports/Netzwerk nicht nutzbar
	ErrBothWindows   = "both_windows"   // Windows↔Windows braucht noch eine Firewall-Regel
)

// Error ist ein Fehler mit einem Code für die Oberfläche.
type Error struct{ Code string }

func (e *Error) Error() string { return e.Code }

// Phase beschreibt, wo die Kopplung gerade steht.
type Phase string

const (
	PhaseIdle      Phase = "idle"      // keine Rolle gewählt
	PhaseSearching Phase = "searching" // sucht bzw. wartet auf die Gegenseite
	PhasePairing   Phase = "pairing"   // Code wird gerade geprüft
	PhaseConnected Phase = "connected" // gekoppelt
)

// Peer ist ein gefundener anderer Rechner.
type Peer struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	OS   string `json:"os"`
	Role Role   `json:"role"`

	addr string // IP:TCP-Port, leer = nimmt keine Verbindungen an
	seen time.Time
}

// State ist der Stand, den die Oberfläche anzeigt.
type State struct {
	Name    string `json:"name"` // eigener Rechnername
	Role    Role   `json:"role"`
	Phase   Phase  `json:"phase"`
	Code    string `json:"code,omitempty"`    // nur beim Empfänger
	Peers   []Peer `json:"peers"`             // passende Gegenstellen
	Partner *Peer  `json:"partner,omitempty"` // nach erfolgreicher Kopplung
	Error   string `json:"error,omitempty"`   // letzter Fehlercode
}

// Config legt fest, wie der Manager das Netz benutzt.
type Config struct {
	Name string // Rechnername
	OS   string // runtime.GOOS

	DiscoveryPort int // UDP; 0 = beliebig (bekommt dann nur Antworten)
	ConnectPort   int // TCP; 0 = beliebig

	// Listen: Verbindungen annehmen. Unter Windows aus – die Firewall würde
	// sie im öffentlichen Netz ohnehin blocken, und so erscheint auch keine
	// Firewall-Abfrage. Windows baut Verbindungen selbst auf (senderDials).
	Listen bool

	// Targets liefert die Ziele für Rundrufe; nil = alle Schnittstellen.
	Targets func() []*net.UDPAddr

	// Logf schreibt Hinweise für die Fehlersuche (z. B. log.Printf); nil = still.
	Logf func(format string, args ...any)
}

func (m *Manager) logf(format string, args ...any) {
	if m.cfg.Logf != nil {
		m.cfg.Logf(format, args...)
	}
}

// DefaultConfig ist die Einstellung für den normalen Betrieb.
func DefaultConfig() Config {
	name, _ := os.Hostname()
	return Config{
		Name:          name,
		OS:            runtime.GOOS,
		DiscoveryPort: DiscoveryPort,
		ConnectPort:   ConnectPort,
		Listen:        runtime.GOOS != "windows",
	}
}

// senderDials sagt, ob der Sender die Verbindung aufbaut. Windows blockt im
// öffentlichen Netz eingehende Verbindungen, ausgehende nicht. Ist genau eine
// Seite Windows, wählt also Windows; sonst wählt der Sender.
func senderDials(senderOS, receiverOS string) bool {
	return !(receiverOS == "windows" && senderOS != "windows")
}

// Manager hält eine Kopplungs-Sitzung zusammen. Alle Methoden sind sicher
// für gleichzeitige Aufrufe.
type Manager struct {
	cfg Config
	id  string
	tls *tls.Config

	mu       sync.Mutex
	role     Role
	phase    Phase
	code     string
	failures int    // falsche Codes in Folge (Empfänger)
	errCode  string // letzter Fehler für die Oberfläche
	peers    map[string]*Peer
	partner  *Peer
	conn     *tls.Conn            // gekoppelte Verbindung
	pending  map[string]*tls.Conn // Sender: Anrufe von Empfängern, warten auf den Code
	dialing  map[string]bool      // Empfänger: laufende Anrufe
	tcpPort  int                  // tatsächlicher TCP-Port, 0 = kein Listener
	udpAddr  *net.UDPAddr         // tatsächliche UDP-Adresse (für Tests)
	ctx      context.Context      // lebt so lange wie die Rolle
	cancel   context.CancelFunc
}

// New erzeugt einen Manager. Er wird erst mit SetRole im Netz aktiv.
func New(cfg Config) (*Manager, error) {
	tc, err := newTLSConfig()
	if err != nil {
		return nil, err
	}
	var id [8]byte
	rand.Read(id[:])
	return &Manager{
		cfg:   cfg,
		id:    hex.EncodeToString(id[:]),
		tls:   tc,
		phase: PhaseIdle,
	}, nil
}

// State liefert eine Kopie des aktuellen Stands.
func (m *Manager) State() State {
	m.mu.Lock()
	defer m.mu.Unlock()
	s := State{Name: m.cfg.Name, Role: m.role, Phase: m.phase, Code: m.code, Error: m.errCode, Peers: []Peer{}}
	if m.partner != nil {
		p := *m.partner
		s.Partner = &p
	}
	want := RoleReceive // der Sender sucht Empfänger und umgekehrt
	if m.role == RoleReceive {
		want = RoleSend
	}
	for _, p := range m.peers {
		if p.Role == want {
			s.Peers = append(s.Peers, *p)
		}
	}
	sort.Slice(s.Peers, func(i, j int) bool { return s.Peers[i].Name < s.Peers[j].Name })
	return s
}

// SetRole wählt die Rolle und beginnt zu suchen bzw. zu warten.
func (m *Manager) SetRole(role Role) error {
	if role != RoleSend && role != RoleReceive {
		return &Error{ErrWrongRole}
	}
	m.Reset()

	ctx, cancel := context.WithCancel(context.Background())
	udp, err := listenUDP(m.cfg.DiscoveryPort)
	if err != nil {
		cancel()
		return &Error{ErrNetwork}
	}
	var ln net.Listener
	if m.cfg.Listen {
		if ln, err = listenTCP(m.cfg.ConnectPort); err != nil {
			udp.Close()
			cancel()
			return &Error{ErrNetwork}
		}
	}

	m.mu.Lock()
	m.role, m.phase, m.errCode = role, PhaseSearching, ""
	m.peers = map[string]*Peer{}
	m.pending = map[string]*tls.Conn{}
	m.dialing = map[string]bool{}
	m.ctx, m.cancel = ctx, cancel
	m.udpAddr = udp.LocalAddr().(*net.UDPAddr)
	m.tcpPort = 0
	if ln != nil {
		m.tcpPort = ln.Addr().(*net.TCPAddr).Port
	}
	if role == RoleReceive {
		m.code, m.failures = newCode(), 0
	}
	tcpInfo := "keiner (baut Verbindungen selbst auf)"
	if m.tcpPort != 0 {
		tcpInfo = strconv.Itoa(m.tcpPort)
	}
	m.logf("Rolle %s: Suche über UDP-Port %d, TCP-Port %s", role, m.udpAddr.Port, tcpInfo)
	m.mu.Unlock()

	if ln != nil {
		go func() { <-ctx.Done(); ln.Close() }()
		go m.acceptLoop(ctx, ln)
	}
	targets := m.cfg.Targets
	if targets == nil {
		targets = func() []*net.UDPAddr { return broadcastTargets(DiscoveryPort) }
	}
	d := &discovery{conn: udp, targets: targets, self: m.beacon, found: m.found}
	go d.run(ctx)
	go m.pruneLoop(ctx)
	return nil
}

// Reset beendet Suche und Verbindung und kehrt zur Rollenwahl zurück.
func (m *Manager) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.cancel != nil {
		m.cancel()
		m.cancel = nil
	}
	if m.conn != nil {
		m.conn.Close()
		m.conn = nil
	}
	for _, c := range m.pending {
		c.Close()
	}
	m.role, m.phase, m.code, m.errCode = "", PhaseIdle, "", ""
	m.peers, m.pending, m.dialing, m.partner = nil, nil, nil, nil
}

// Pair koppelt den Sender mit dem Empfänger id per Code.
func (m *Manager) Pair(ctx context.Context, id, code string) error {
	code = strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, code)
	if len(code) != 6 {
		return &Error{ErrBadCode}
	}

	m.mu.Lock()
	if m.role != RoleSend || m.phase != PhaseSearching {
		m.mu.Unlock()
		return &Error{ErrWrongRole}
	}
	p := m.peers[id]
	if p == nil {
		m.mu.Unlock()
		return &Error{ErrPeerGone}
	}
	peer := *p
	m.phase, m.errCode = PhasePairing, ""
	m.mu.Unlock()

	c, err := m.pairAsSender(ctx, &peer, code)
	if err == nil && m.paired(c, &peer) {
		return nil
	}
	if c != nil {
		c.Close()
	}
	ec := ErrConnectFailed
	var e *Error
	if errors.As(err, &e) {
		ec = e.Code
	} else if errors.Is(err, errWrongCode) {
		ec = ErrWrongCode
	}
	m.mu.Lock()
	if m.phase == PhasePairing {
		m.phase = PhaseSearching
	}
	m.errCode = ec
	m.mu.Unlock()
	return &Error{ec}
}

func (m *Manager) pairAsSender(ctx context.Context, p *Peer, code string) (*tls.Conn, error) {
	if senderDials(m.cfg.OS, p.OS) {
		if p.addr == "" {
			if m.cfg.OS == "windows" && p.OS == "windows" {
				// TODO(Meilenstein 2): Firewall-Regel per UAC anlegen.
				return nil, &Error{ErrBothWindows}
			}
			return nil, &Error{ErrConnectFailed}
		}
		c, err := m.dial(ctx, p.addr)
		if err != nil {
			return nil, err
		}
		c.SetDeadline(time.Now().Add(30 * time.Second))
		err = pakeClient(c, code)
		c.SetDeadline(time.Time{})
		return c, err
	}
	// Der Empfänger (Windows) ruft selbst an; auf diesen Anruf warten.
	c, err := m.takePending(ctx, p.ID, 15*time.Second)
	if err != nil {
		return nil, err
	}
	c.SetDeadline(time.Now().Add(30 * time.Second))
	err = pakeServer(c, code)
	c.SetDeadline(time.Time{})
	return c, err
}

func (m *Manager) takePending(ctx context.Context, id string, wait time.Duration) (*tls.Conn, error) {
	deadline := time.Now().Add(wait)
	for time.Now().Before(deadline) {
		m.mu.Lock()
		c := m.pending[id]
		delete(m.pending, id)
		m.mu.Unlock()
		if c != nil {
			return c, nil
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(100 * time.Millisecond):
		}
	}
	m.mu.Lock()
	port := m.tcpPort
	m.mu.Unlock()
	m.logf("Der Empfänger hat nicht angerufen (blockt hier eine Firewall TCP-Port %d?)", port)
	return nil, &Error{ErrConnectFailed}
}

// paired übernimmt eine erfolgreich gekoppelte Verbindung.
func (m *Manager) paired(c *tls.Conn, p *Peer) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.phase == PhaseConnected || m.role == "" {
		return false // schon verbunden oder inzwischen abgebrochen
	}
	partner := *p
	m.conn, m.partner, m.phase, m.errCode = c, &partner, PhaseConnected, ""
	m.logf("Gekoppelt mit %s (%s)", p.Name, p.OS)
	// Suche beenden; die Verbindung selbst bleibt bestehen.
	m.cancel()
	m.cancel = nil
	for id, pc := range m.pending {
		pc.Close()
		delete(m.pending, id)
	}
	go m.watch(c)
	return true
}

// watch merkt, wenn die Gegenseite die Verbindung trennt.
// TODO(Meilenstein 2): Durch das Übertragungsprotokoll ersetzen.
func (m *Manager) watch(c *tls.Conn) {
	for {
		if _, err := readFrame(c); err != nil {
			break
		}
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.conn == c {
		m.conn, m.partner, m.role, m.phase, m.code = nil, nil, "", PhaseIdle, ""
		m.errCode = ErrPeerGone
	}
}

func (m *Manager) beacon() beacon {
	m.mu.Lock()
	defer m.mu.Unlock()
	return beacon{App: "flohh", V: protoVersion, ID: m.id, Name: m.cfg.Name, OS: m.cfg.OS, Role: m.role, Port: m.tcpPort}
}

func (m *Manager) helloMsg() hello {
	m.mu.Lock()
	defer m.mu.Unlock()
	return hello{ID: m.id, Name: m.cfg.Name, OS: m.cfg.OS, Role: m.role}
}

// found wird für jedes empfangene Rundruf-Paket aufgerufen.
func (m *Manager) found(b beacon, from *net.UDPAddr) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.phase != PhaseSearching && m.phase != PhasePairing {
		return
	}
	p := &Peer{ID: b.ID, Name: b.Name, OS: b.OS, Role: b.Role, seen: time.Now()}
	if b.Port != 0 {
		p.addr = net.JoinHostPort(from.IP.String(), strconv.Itoa(b.Port))
	}
	if _, known := m.peers[b.ID]; !known {
		addr := p.addr
		if addr == "" {
			addr = from.IP.String() + " (nimmt keine Verbindungen an)"
		}
		m.logf("Gefunden: %s (%s, Rolle %s) unter %s", b.Name, b.OS, b.Role, addr)
	}
	m.peers[b.ID] = p

	// Windows-Empfänger: Der Sender kann uns nicht erreichen, also rufen wir
	// ihn an. Die Kopplung wartet dann, bis dort der Code eingetippt ist.
	if m.role == RoleReceive && b.Role == RoleSend && !senderDials(b.OS, m.cfg.OS) &&
		p.addr != "" && !m.dialing[b.ID] {
		m.dialing[b.ID] = true
		go m.dialAsReceiver(m.ctx, *p)
	}
}

func (m *Manager) dialAsReceiver(ctx context.Context, p Peer) {
	defer func() {
		m.mu.Lock()
		delete(m.dialing, p.ID) // beim nächsten Rundruf erneut versuchen
		m.mu.Unlock()
	}()
	c, err := m.dial(ctx, p.addr)
	if err != nil {
		return
	}
	// Abbruch (Reset) beendet das Warten auf den Code. Nach der Kopplung darf
	// das nicht mehr greifen – paired() beendet den Kontext ja absichtlich.
	stop := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			c.Close()
		case <-stop:
		}
	}()
	m.mu.Lock()
	code := m.code
	m.mu.Unlock()
	err = pakeClient(c, code)
	close(stop)
	if err == nil && m.paired(c, &p) {
		return
	}
	c.Close()
	m.countFailure(err)
}

func (m *Manager) acceptLoop(ctx context.Context, ln net.Listener) {
	for {
		raw, err := ln.Accept()
		if err != nil {
			return
		}
		go m.handleIncoming(ctx, raw)
	}
}

func (m *Manager) handleIncoming(ctx context.Context, raw net.Conn) {
	m.logf("Eingehende Verbindung von %s", raw.RemoteAddr())
	c := tls.Server(raw, m.tls)
	c.SetDeadline(time.Now().Add(10 * time.Second))
	if err := c.HandshakeContext(ctx); err != nil {
		m.logf("Verschlüsselung mit %s fehlgeschlagen: %v", raw.RemoteAddr(), err)
		c.Close()
		return
	}
	h, err := readHello(c)
	c.SetDeadline(time.Time{})
	if err != nil {
		c.Close()
		return
	}
	p := Peer{ID: h.ID, Name: h.Name, OS: h.OS, Role: h.Role}

	m.mu.Lock()
	role, code := m.role, m.code
	m.mu.Unlock()

	switch {
	case role == RoleReceive && h.Role == RoleSend:
		// Der Sender hat den Code schon eingetippt und ruft an.
		c.SetDeadline(time.Now().Add(30 * time.Second))
		err := pakeServer(c, code)
		c.SetDeadline(time.Time{})
		if err == nil && m.paired(c, &p) {
			return
		}
		c.Close()
		m.countFailure(err)
	case role == RoleSend && h.Role == RoleReceive:
		// Ein Windows-Empfänger ruft an; wartet auf den Code (Pair).
		m.mu.Lock()
		if m.pending == nil {
			m.mu.Unlock()
			c.Close()
			return
		}
		if old := m.pending[h.ID]; old != nil {
			old.Close()
		}
		m.pending[h.ID] = c
		m.mu.Unlock()
	default:
		c.Close()
	}
}

// countFailure zählt falsche Codes beim Empfänger. Nach drei Fehlversuchen
// gibt es einen neuen Code – so bleibt Raten aussichtslos.
func (m *Manager) countFailure(err error) {
	if !errors.Is(err, errWrongCode) {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.role != RoleReceive || m.phase == PhaseConnected {
		return
	}
	m.errCode = ErrWrongCode
	m.failures++
	if m.failures >= 3 {
		m.code, m.failures = newCode(), 0
	}
}

func (m *Manager) dial(ctx context.Context, addr string) (*tls.Conn, error) {
	m.logf("Verbinde mit %s …", addr)
	d := net.Dialer{Timeout: 5 * time.Second}
	raw, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		m.logf("Verbindung zu %s fehlgeschlagen: %v (blockt dort eine Firewall?)", addr, err)
		return nil, &Error{ErrConnectFailed}
	}
	c := tls.Client(raw, m.tls)
	c.SetDeadline(time.Now().Add(10 * time.Second))
	if err := c.HandshakeContext(ctx); err != nil {
		m.logf("Verschlüsselung mit %s fehlgeschlagen: %v", addr, err)
		c.Close()
		return nil, &Error{ErrConnectFailed}
	}
	if err := writeHello(c, m.helloMsg()); err != nil {
		m.logf("Verbindung zu %s abgebrochen: %v", addr, err)
		c.Close()
		return nil, &Error{ErrConnectFailed}
	}
	m.logf("Verbunden mit %s, warte auf Kopplung per Code", addr)
	c.SetDeadline(time.Time{})
	return c, nil
}

// pruneLoop entfernt Gegenstellen, die sich länger nicht gemeldet haben.
func (m *Manager) pruneLoop(ctx context.Context) {
	t := time.NewTicker(time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
		m.mu.Lock()
		for id, p := range m.peers {
			if time.Since(p.seen) > 4*time.Second {
				delete(m.peers, id)
			}
		}
		m.mu.Unlock()
	}
}

// listenUDP öffnet den Rundruf-Port. Ist er belegt (z. B. zweite Instanz auf
// demselben Rechner), nimmt sie einen beliebigen – dann kommen nur Antworten
// an, das reicht zum Finden.
func listenUDP(port int) (*net.UDPConn, error) {
	c, err := net.ListenUDP("udp4", &net.UDPAddr{Port: port})
	if err != nil && port != 0 {
		c, err = net.ListenUDP("udp4", &net.UDPAddr{})
	}
	return c, err
}

func listenTCP(port int) (net.Listener, error) {
	ln, err := net.Listen("tcp4", ":"+strconv.Itoa(port))
	if err != nil && port != 0 {
		ln, err = net.Listen("tcp4", ":0")
	}
	return ln, err
}
