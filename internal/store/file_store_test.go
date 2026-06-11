package store

import "testing"

func TestFileStoreAddListDelete(t *testing.T) {
	store, err := NewFileStore(t.TempDir(), 1<<20)
	if err != nil {
		t.Fatal(err)
	}

	item, err := store.Add("main", "", "hello", "https://example.com", nil)
	if err != nil {
		t.Fatal(err)
	}

	items, err := store.List("main")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].ID != item.ID || items[0].Text != "hello" || items[0].URL != "https://example.com" {
		t.Fatalf("unexpected item: %+v", items[0])
	}

	if otherItems, err := store.List("second"); err != nil || len(otherItems) != 0 {
		t.Fatalf("unexpected other user items: %+v err=%v", otherItems, err)
	}

	if err := store.Delete("main", item.ID); err != nil {
		t.Fatal(err)
	}
	items, err = store.List("main")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Fatal("item was not deleted")
	}
}

func TestFileStoreMovesInvalidURLToText(t *testing.T) {
	store, err := NewFileStore(t.TempDir(), 1<<20)
	if err != nil {
		t.Fatal(err)
	}

	item, err := store.Add("main", "", "prefix", "not a url", nil)
	if err != nil {
		t.Fatal(err)
	}
	if item.URL != "" {
		t.Fatalf("expected empty url, got %q", item.URL)
	}
	if item.Text != "prefix\nnot a url" {
		t.Fatalf("unexpected text: %q", item.Text)
	}
}
