package envfile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "nexus-mini.env")
	body := "# тайлбар\n\nA=1\n B = 2 \nC=\"хашилттай\"\nD=a=b=c\nБуруу мөр\nE=\n"
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("A", "")
	t.Setenv("B", "")
	t.Setenv("C", "")
	t.Setenv("D", "")
	t.Setenv("E", "")
	t.Setenv("PRESET", "хэвээр")
	if err := Load(p); err != nil {
		t.Fatal(err)
	}
	for k, want := range map[string]string{"A": "1", "B": "2", "C": "хашилттай", "D": "a=b=c", "E": ""} {
		if got := os.Getenv(k); got != want {
			t.Errorf("%s = %q, хүлээсэн %q", k, got, want)
		}
	}
	// Орчны хувьсагч файлаас дээгүүр.
	if err := os.WriteFile(p, []byte("PRESET=файлаас\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := Load(p); err != nil {
		t.Fatal(err)
	}
	if os.Getenv("PRESET") != "хэвээр" {
		t.Fatal("файл орчны хувьсагчийг дарав")
	}
}

func TestLoadMissingFileIsNotAnError(t *testing.T) {
	if err := Load(filepath.Join(t.TempDir(), "байхгүй.env")); err != nil {
		t.Fatalf("байхгүй файл алдаа өглөө: %v", err)
	}
}
