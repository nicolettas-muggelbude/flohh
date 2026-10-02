// mkusbimg erzeugt aus einem Ordner ein Stick-Abbild (.img): MBR-Partitions-
// tabelle mit einer FAT32-Partition, in der alle Dateien des Ordners liegen.
//
// Das Abbild wird mit Rufus, balenaEtcher, Raspberry Pi Imager, GNOME
// "Laufwerke" oder dd auf einen USB-Stick geschrieben. Anders als bei einem
// ISO bleibt der Stick danach beschreibbar.
//
// Läuft ohne Adminrechte und ohne Systemwerkzeuge (reines Go).
//
//	go run ./mkusbimg -in ../dist/usb -out ../dist/umzugskarton-usb.img
package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	diskfs "github.com/diskfs/go-diskfs"
	"github.com/diskfs/go-diskfs/disk"
	"github.com/diskfs/go-diskfs/filesystem"
	"github.com/diskfs/go-diskfs/partition/mbr"
)

const (
	sectorSize     = 512
	partitionStart = 2048             // erste Partition bei 1 MiB, wie bei üblichen Werkzeugen
	minImageSize   = 64 * 1024 * 1024 // FAT32 braucht eine gewisse Mindestgröße
	mib            = 1024 * 1024
)

func main() {
	in := flag.String("in", "", "Ordner, dessen Dateien auf den Stick sollen")
	out := flag.String("out", "", "Ziel-Datei (.img)")
	label := flag.String("label", "UMZUG", "Name des Sticks (max. 11 Zeichen)")
	flag.Parse()
	if *in == "" || *out == "" {
		flag.Usage()
		os.Exit(2)
	}
	if err := run(*in, *out, *label); err != nil {
		log.Fatal(err)
	}
}

func run(in, out, label string) error {
	entries, err := os.ReadDir(in)
	if err != nil {
		return err
	}

	// Größe: Inhalt + 25 % Reserve für FAT-Verwaltung und spätere Berichte,
	// auf ganze MiB aufgerundet, mindestens minImageSize.
	var content int64
	for _, e := range entries {
		if e.IsDir() {
			return fmt.Errorf("%s: Unterordner werden nicht unterstützt", e.Name())
		}
		info, err := e.Info()
		if err != nil {
			return err
		}
		content += info.Size()
	}
	size := max(minImageSize, (content*5/4/mib+1)*mib+partitionStart*sectorSize)

	// Ein vorhandenes Abbild wird ersetzt – es ist ein Build-Ergebnis.
	if err := os.Remove(out); err != nil && !os.IsNotExist(err) {
		return err
	}
	img, err := diskfs.Create(out, size, diskfs.SectorSize512)
	if err != nil {
		return err
	}
	// Bei einem Fehler unterwegs schließen; im Erfolgsfall wird unten
	// ausdrücklich geschlossen (ein zweites Close stürzt in go-diskfs ab).
	closed := false
	defer func() {
		if !closed {
			img.Close()
		}
	}()

	table := &mbr.Table{
		LogicalSectorSize:  sectorSize,
		PhysicalSectorSize: sectorSize,
		Partitions: []*mbr.Partition{{
			Index: 1,
			Type:  mbr.Fat32LBA,
			Start: partitionStart,
			Size:  uint32(size/sectorSize) - partitionStart,
		}},
	}
	if err := img.Partition(table); err != nil {
		return fmt.Errorf("Partitionstabelle: %w", err)
	}

	fs, err := img.CreateFilesystem(disk.FilesystemSpec{
		Partition:   1,
		FSType:      filesystem.TypeFat32,
		VolumeLabel: label,
	})
	if err != nil {
		return fmt.Errorf("FAT32 anlegen: %w", err)
	}

	for _, e := range entries {
		if err := copyFile(fs, filepath.Join(in, e.Name()), "/"+e.Name()); err != nil {
			return fmt.Errorf("%s: %w", e.Name(), err)
		}
	}
	closed = true
	if err := img.Close(); err != nil {
		return err
	}

	if err := fixHeaders(out); err != nil {
		return fmt.Errorf("Kopfdaten: %w", err)
	}
	if err := verify(out, in, entries); err != nil {
		return fmt.Errorf("Prüfung: %w", err)
	}

	fmt.Printf("%s: %d MiB, %d Dateien, geprüft\n", out, size/mib, len(entries))
	return nil
}

// fixHeaders ergänzt zwei Werte, die go-diskfs leer lässt, auf die Windows
// aber Wert legt:
//   - Datenträger-ID im MBR (Byte 440–443): darf nicht 0 sein.
//   - "Hidden Sectors" im FAT32-Bootsektor und seiner Sicherungskopie
//     (Byte 28–31): Startsektor der Partition.
func fixHeaders(path string) error {
	f, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		return err
	}
	defer f.Close()

	id := make([]byte, 4)
	if _, err := rand.Read(id); err != nil {
		return err
	}
	id[0] |= 1 // garantiert ungleich 0
	if _, err := f.WriteAt(id, 440); err != nil {
		return err
	}

	boot := int64(partitionStart * sectorSize)
	// Wo die Sicherungskopie des Bootsektors liegt, steht im Bootsektor selbst
	// (Byte 50–51, in Sektoren ab Partitionsbeginn; üblich ist 6).
	b := make([]byte, 2)
	if _, err := f.ReadAt(b, boot+50); err != nil {
		return err
	}
	backup := boot + int64(binary.LittleEndian.Uint16(b))*sectorSize

	hidden := binary.LittleEndian.AppendUint32(nil, partitionStart)
	for _, at := range []int64{boot, backup} {
		if _, err := f.WriteAt(hidden, at+28); err != nil {
			return err
		}
	}
	return f.Sync()
}

// verify öffnet das fertige Abbild erneut und vergleicht jede Datei per
// SHA-256 mit dem Original.
func verify(path, in string, entries []os.DirEntry) error {
	img, err := diskfs.Open(path, diskfs.WithOpenMode(diskfs.ReadOnly))
	if err != nil {
		return err
	}
	defer img.Close()
	fs, err := img.GetFilesystem(1)
	if err != nil {
		return err
	}
	for _, e := range entries {
		want, err := os.ReadFile(filepath.Join(in, e.Name()))
		if err != nil {
			return err
		}
		got, err := fs.ReadFile(e.Name())
		if err != nil {
			return fmt.Errorf("%s: %w", e.Name(), err)
		}
		if sha256.Sum256(got) != sha256.Sum256(want) {
			return fmt.Errorf("%s: Inhalt weicht ab", e.Name())
		}
	}
	return nil
}

func copyFile(fs filesystem.FileSystem, src, dst string) error {
	r, err := os.Open(src)
	if err != nil {
		return err
	}
	defer r.Close()

	w, err := fs.OpenFile(dst, os.O_CREATE|os.O_RDWR)
	if err != nil {
		return err
	}
	if _, err := io.Copy(w, r); err != nil {
		w.Close()
		return err
	}
	return w.Close()
}
