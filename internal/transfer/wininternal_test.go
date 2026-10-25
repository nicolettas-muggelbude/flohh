package transfer

import (
	"path/filepath"
	"testing"
	"time"
)

func TestWindowsInternal(t *testing.T) {
	cases := map[string]bool{
		`C:\Users\Nicole\AppData\Local\Temp\setup.log`:                                  true,
		`C:\Users\Nicole\AppData\Local\Microsoft\Windows\UsrClass.dat`:                  true,
		`C:\Users\Nicole\AppData\Local\Microsoft\Windows\WebCache\WebCacheV01.dat`:      true,
		`C:\Users\Nicole\AppData\Local\Microsoft\Windows\Explorer\thumbcache_256.db`:    true,
		`C:\Users\Nicole\AppData\Roaming\Microsoft\Windows\Recent\Brief.lnk`:            true,
		`C:\Users\Nicole\AppData\Local\Packages\Microsoft.Photos_8wekyb3d8bbwe\AC\x.db`: true,
		`C:\Users\Nicole\NTUSER.DAT`:                                                    true,
		`C:\Users\Nicole\ntuser.dat.LOG1`:                                               true,
		// bleibt:
		`C:\Users\Nicole\AppData\Local\Microsoft\Windows\Fonts\MeineSchrift.ttf`:                                     false,
		`C:\Users\Nicole\AppData\Roaming\Microsoft\Signatures\Arbeit.htm`:                                            false,
		`C:\Users\Nicole\AppData\Roaming\Thunderbird\Profiles\x\Mail\INBOX`:                                          false,
		`C:\Users\Nicole\AppData\Local\Packages\Microsoft.MicrosoftStickyNotes_8wekyb3d8bbwe\LocalState\plum.sqlite`: false,
		`C:\Users\Nicole\Documents\Temp\notiz.txt`:                                                                   false,
		"/home/nicole/.config/app.conf":                                                                              false,
	}
	for p, want := range cases {
		if got := windowsInternal(p); got != want {
			t.Errorf("windowsInternal(%q) = %v, erwartet %v", p, got, want)
		}
	}
}

// Ist AppData selbst freigegeben, werden Windows-interne Dateien gezählt,
// aber nicht übertragen.
func TestWindowsInternalSkipped(t *testing.T) {
	tmp := t.TempDir()
	if windowsInternal(tmp + "/") {
		// Unter Windows liegt der Testordner selbst in AppData\Local\Temp –
		// dann filtert flohh absichtlich nichts (bewusst gewählter Ordner).
		t.Skip("Testordner liegt in einem Windows-internen Bereich")
	}
	root := filepath.Join(tmp, "AppData")
	write(t, filepath.Join(root, "Local", "Temp", "x.tmp"), []byte("x"), time.Now())
	write(t, filepath.Join(root, "Local", "Microsoft", "Windows", "UsrClass.dat"), []byte("x"), time.Now())
	write(t, filepath.Join(root, "Roaming", "Thunderbird", "profiles.ini"), []byte("x"), time.Now())

	st := Count(root)
	if st.Files != 1 || st.System != 2 {
		t.Errorf("Zählung: %+v", st)
	}

	// Bewusst gewählter Ordner, der selbst Windows-intern ist: alles kommt mit.
	if st := Count(filepath.Join(root, "Local", "Temp")); st.Files != 1 || st.System != 0 {
		t.Errorf("Temp bewusst freigegeben: %+v", st)
	}
}
