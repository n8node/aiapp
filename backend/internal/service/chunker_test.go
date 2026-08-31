package service

import "testing"

func TestChunkTextShort(t *testing.T) {
	got := chunkText("привет мир", 500, 50)
	if len(got) != 1 || got[0].Text != "привет мир" {
		t.Fatalf("%#v", got)
	}
}

func TestChunkTextSplits(t *testing.T) {
	var b []rune
	for i := 0; i < 1200; i++ {
		if i > 0 && i%80 == 0 {
			b = append(b, '\n', '\n')
		} else {
			b = append(b, 'а')
		}
	}
	got := chunkText(string(b), 200, 20)
	if len(got) < 4 {
		t.Fatalf("got %d chunks", len(got))
	}
	if got[0].Index != 0 || got[1].Index != 1 {
		t.Fatal("index")
	}
}

func TestChunkTextEmpty(t *testing.T) {
	if chunkText("   ", 100, 10) != nil {
		t.Fatal("empty")
	}
}

func TestUniqueIDs(t *testing.T) {
	got := uniqueIDs([]string{" a ", "a", "", "b", "b"})
	if len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("%#v", got)
	}
}
