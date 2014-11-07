package messaging

import (
	"reflect"
	"testing"
)

func TestAddressMarshaling(t *testing.T) {
	orig := Address(randomWendyID())
	bytes, err := orig.MarshalBinary()
	if err != nil {
		t.Fatal("Couldn't marshal address:", err)
	}

	unm := new(Address)
	err = unm.UnmarshalBinary(bytes)
	t.Logf("bytes %#v, unm %#v", bytes, unm)
	if err != nil {
		t.Fatal("Couldn't unmarshal message:", err)
	}
	if !reflect.DeepEqual(orig, *unm) {
		t.Errorf("orig %#v != %#v", orig, *unm)
	}
}
func TestMessageEncoding(t *testing.T) {
	orig := &Message{Version: 1, From: Address(randomWendyID()), To: Address(randomWendyID()), Data: []byte{1, 1, 2, 3, 5, 8}}
	bytes, err := orig.GobEncode()
	if err != nil {
		t.Fatal("Couldn't marshal message:", err)
	}

	var unm *Message
	err = unm.GobDecode(bytes)
	if err != nil {
		t.Fatal("Couldn't unmarshal message:", err)
	}
	if !reflect.DeepEqual(orig, unm) {
		t.Errorf("orig %#v != %#v", orig, unm)
	}
}
