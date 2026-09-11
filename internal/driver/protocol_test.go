package driver

import "testing"

func TestSplitLine(t *testing.T) {
	tokens, err := splitLine(`SETINFO device.model "HID UPS Battery"`)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"SETINFO", "device.model", "HID UPS Battery"}
	if len(tokens) != len(want) {
		t.Fatalf("got %#v", tokens)
	}
	for i := range want {
		if tokens[i] != want[i] {
			t.Fatalf("token %d: got %q want %q", i, tokens[i], want[i])
		}
	}
}

func TestSourceApply(t *testing.T) {
	store := stateForTest()
	source := Source{Store: store}
	source.Store.Connected()
	if err := source.apply(`SETINFO ups.status "OL"`); err != nil {
		t.Fatal(err)
	}
	if err := source.apply("DATAOK"); err != nil {
		t.Fatal(err)
	}
	snap := store.Snapshot()
	if snap.Variables["ups.status"] != "OL" || !snap.DataOK {
		t.Fatalf("unexpected snapshot: %#v", snap)
	}
}
