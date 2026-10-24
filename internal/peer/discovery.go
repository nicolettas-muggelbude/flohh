// Package peer findet andere flohh-Instanzen im Netz und koppelt sich mit
// genau einer davon – verschlüsselt und abgesichert durch einen 6-stelligen
// Code, den der Empfänger anzeigt und der Sender eintippt.
package peer

import (
	"context"
	"encoding/json"
	"net"
	"time"
)

// Feste Ports, damit sich Rechner ohne Absprache finden. Später braucht die
// Windows-Firewall-Regel (Windows↔Windows) genau diese Ports.
const (
	DiscoveryPort = 41730 // UDP: Rundrufe zum Finden
	ConnectPort   = 41731 // TCP: die eigentliche Verbindung
)

const protoVersion = 1

// Role ist die Rolle, die jemand in der Oberfläche gewählt hat.
type Role string

const (
	RoleSend    Role = "send"    // alter Rechner, gibt Daten frei
	RoleReceive Role = "receive" // neuer Rechner, holt Daten ab
)

// beacon ist die Nachricht, mit der sich flohh-Instanzen per UDP melden.
type beacon struct {
	App  string `json:"app"`  // immer "flohh" – fremde Pakete werden ignoriert
	V    int    `json:"v"`    // Protokollversion
	Type string `json:"type"` // "hello" (Rundruf) oder "here" (Antwort)
	ID   string `json:"id"`   // zufällig pro Programmstart
	Name string `json:"name"` // Rechnername
	OS   string `json:"os"`   // runtime.GOOS
	Role Role   `json:"role"`
	Port int    `json:"port"` // TCP-Port für Verbindungen, 0 = nimmt keine an
}

// discovery schickt jede Sekunde einen Rundruf und beantwortet fremde.
//
// Warum das auch mit Windows im „öffentlichen“ Netz klappt: Die Firewall
// blockt unaufgeforderte Pakete, erlaubt aber Antworten auf einen eigenen
// Rundruf. Windows ruft, die Gegenseite antwortet direkt – so lernt auch
// Windows die Adresse der Gegenseite.
type discovery struct {
	conn    *net.UDPConn
	targets func() []*net.UDPAddr
	self    func() beacon
	found   func(b beacon, from *net.UDPAddr)
}

func (d *discovery) run(ctx context.Context) {
	go d.readLoop()
	t := time.NewTicker(time.Second)
	defer t.Stop()
	for {
		d.sendHello()
		select {
		case <-ctx.Done():
			d.conn.Close() // beendet auch readLoop
			return
		case <-t.C:
		}
	}
}

func (d *discovery) sendHello() {
	b := d.self()
	b.Type = "hello"
	data, err := json.Marshal(b)
	if err != nil {
		return
	}
	for _, a := range d.targets() {
		// Fehler sind normal: Nicht über jede Schnittstelle lässt sich senden.
		d.conn.WriteToUDP(data, a)
	}
}

func (d *discovery) readLoop() {
	buf := make([]byte, 2048)
	for {
		n, from, err := d.conn.ReadFromUDP(buf)
		if err != nil {
			return // Socket geschlossen
		}
		var b beacon
		if json.Unmarshal(buf[:n], &b) != nil || b.App != "flohh" || b.V != protoVersion {
			continue
		}
		me := d.self()
		if b.ID == me.ID {
			continue // eigener Rundruf
		}
		if b.Type == "hello" {
			me.Type = "here"
			if data, err := json.Marshal(me); err == nil {
				d.conn.WriteToUDP(data, from)
			}
		}
		d.found(b, from)
	}
}

// multicastTargets6 liefert für jede aktive Schnittstelle die IPv6-Adresse
// „alle Geräte an dieser Leitung“ (ff02::1). Darüber finden sich zwei
// Rechner am selben Kabel immer – egal, wie IPv4 eingestellt ist.
func multicastTargets6(port int) []*net.UDPAddr {
	var out []*net.UDPAddr
	ifaces, _ := net.Interfaces()
	for _, ifc := range ifaces {
		if ifc.Flags&net.FlagUp == 0 || ifc.Flags&net.FlagMulticast == 0 || ifc.Flags&net.FlagLoopback != 0 {
			continue
		}
		out = append(out, &net.UDPAddr{IP: net.IPv6linklocalallnodes, Port: port, Zone: ifc.Name})
	}
	return out
}

// broadcastTargets liefert die Rundruf-Adressen aller aktiven Schnittstellen.
// Dazu gehört bei direktem Kabel ohne Router auch 169.254.255.255.
func broadcastTargets(port int) []*net.UDPAddr {
	seen := map[string]bool{}
	var out []*net.UDPAddr
	add := func(ip net.IP) {
		if k := ip.String(); !seen[k] {
			seen[k] = true
			out = append(out, &net.UDPAddr{IP: ip, Port: port})
		}
	}
	add(net.IPv4bcast)

	ifaces, _ := net.Interfaces()
	for _, ifc := range ifaces {
		if ifc.Flags&net.FlagUp == 0 || ifc.Flags&net.FlagBroadcast == 0 || ifc.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, _ := ifc.Addrs()
		for _, a := range addrs {
			n, ok := a.(*net.IPNet)
			if !ok {
				continue
			}
			ip4 := n.IP.To4()
			mask := n.Mask
			if len(mask) == net.IPv6len {
				mask = mask[12:]
			}
			if ip4 == nil || len(mask) != net.IPv4len {
				continue
			}
			b := make(net.IP, net.IPv4len)
			for i := range b {
				b[i] = ip4[i] | ^mask[i]
			}
			add(b)
		}
	}
	return out
}
