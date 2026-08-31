package archivekit

import (
	"archive/zip"
	"bytes"
	"testing"
)

func TestArchiveBaseName(t *testing.T) {
	if got := ArchiveBaseName("contracts.zip"); got != "contracts" {
		t.Fatalf("got %q", got)
	}
	if got := ArchiveBaseName("pack.tar.gz"); got != "pack" {
		t.Fatalf("tgz %q", got)
	}
	if got := ArchiveBaseName("  .zip"); got != "архив" && ArchiveBaseName(".zip") == "" {
		t.Fatalf("empty")
	}
}

func TestNextAvailableName(t *testing.T) {
	taken := map[string]bool{"Договоры": true, "Договоры (1)": true}
	got, err := NextAvailableName("Договоры", func(s string) bool { return taken[s] })
	if err != nil || got != "Договоры (2)" {
		t.Fatalf("got %q %v", got, err)
	}
	free, err := NextAvailableName("Новая", func(string) bool { return false })
	if err != nil || free != "Новая" {
		t.Fatalf("free %q", free)
	}
}

func TestSafeRelPath(t *testing.T) {
	if _, err := SafeRelPath("../etc/passwd"); err == nil {
		t.Fatal("zip slip")
	}
	if _, err := SafeRelPath("/abs/path.txt"); err != nil {
		t.Fatal("rooted path should clean")
	}
	if _, err := SafeRelPath(`..\windows\system32`); err == nil {
		t.Fatal("backslash slip")
	}
	got, err := SafeRelPath("folder/inner/a.pdf")
	if err != nil || got != "folder/inner/a.pdf" {
		t.Fatalf("ok path %q %v", got, err)
	}
}

func TestWalkZipSkipsExeAndSlip(t *testing.T) {
	buf := new(bytes.Buffer)
	zw := zip.NewWriter(buf)
	w, err := zw.Create("../evil.txt")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = w.Write([]byte("nope"))
	w, err = zw.Create("ok.txt")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = w.Write([]byte("hello"))
	w, err = zw.Create("bad.exe")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = w.Write([]byte("MZ"))
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	entries, err := Walk("pack.zip", buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].RelPath != "ok.txt" {
		t.Fatalf("entries %#v", entries)
	}
}
