package filesniff

import "testing"

func TestExtensionAndAllow(t *testing.T) {
	if Extension("Report.PDF") != "pdf" || !AllowedExtension("a.docx") {
		t.Fatal("ext")
	}
	if AllowedExtension("a.exe") || AllowedExtension("noext") {
		t.Fatal("reject")
	}
}

func TestHeadMatches(t *testing.T) {
	if !HeadMatches("a.pdf", []byte("%PDF-1.7 rest")) {
		t.Fatal("pdf")
	}
	if HeadMatches("a.pdf", []byte("MZ")) {
		t.Fatal("exe as pdf")
	}
	if !HeadMatches("a.docx", []byte("PK\x03\x04hello")) {
		t.Fatal("docx zip")
	}
	if !HeadMatches("note.txt", []byte("hello\nworld")) {
		t.Fatal("txt")
	}
	if HeadMatches("note.txt", []byte{0, 1, 2, 3}) {
		t.Fatal("binary as txt")
	}
}

func TestMIMEMatchesExtension(t *testing.T) {
	if !MIMEMatchesExtension("application/pdf", "x.pdf") {
		t.Fatal("pdf mime")
	}
	if MIMEMatchesExtension("application/pdf", "x.jpg") {
		t.Fatal("mismatch")
	}
	if !MIMEMatchesExtension("application/octet-stream", "x.pdf") {
		t.Fatal("generic mime ok")
	}
}
