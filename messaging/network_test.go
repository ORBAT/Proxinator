package messaging

import (
	"fmt"
	"net"
	"reflect"
	"strconv"
	"sync"

	"github.com/bradfitz/iter"

	"testing"
	"time"

	"github.com/ORBAT/wendy"
)

type mockCluster struct {
	sent chan *wendy.Message
	app  wendy.Application
}

func newMockCluster(sentCh chan *wendy.Message, app wendy.Application) *mockCluster {
	return &mockCluster{sentCh, app}
}

func (mc *mockCluster) Send(wm wendy.Message) error {
	go func() {
		mc.sent <- &wm
	}()
	return nil
}

func (mc *mockCluster) NewMessage(purpose byte, nid wendy.NodeID, val []byte) wendy.Message {
	return wendy.Message{Purpose: purpose, Key: nid, Value: val}
}

func (mc *mockCluster) Stop() {}

func newMockConf() *Config {
	return &Config{Address: Address(randomWendyID())}
}

func newMockNode(a Address) (*Node, <-chan *wendy.Message) {
	conf := newMockConf()
	nd := conf.initNode()

	sentCh := make(chan *wendy.Message, 20)
	app := newWendyApp(a)
	mc := newMockCluster(sentCh, app)

	nd.wcluster = mc
	nd.wapp = app
	return nd, sentCh
}

func ok(v interface{}) {
	if v != nil {
		panic(v)
	}
}

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
	orig := &Message{Version: ProtocolVersion, From: Address(randomWendyID()), To: Address(randomWendyID()), Data: []byte{1, 1, 2, 3, 5, 8}}
	bytes, err := orig.Encode()
	if err != nil {
		t.Fatal("Couldn't marshal message:", err)
	}

	unm := new(Message)
	err = unm.Decode(bytes)
	if err != nil {
		t.Fatal("Couldn't unmarshal message:", err)
	}
	if !reflect.DeepEqual(orig, unm) {
		t.Errorf("orig %#v != %#v", orig, unm)
	}
}

func TestMsgFromWendy(t *testing.T) {
	from, err := NewAddress([]byte("1111111111111111"))
	ok(err)
	to, err := NewAddress([]byte("2222222222222222"))
	ok(err)

	orig := &Message{Version: ProtocolVersion,
		From: from,
		To:   to,
		Data: []byte("DATS SUM PAYLOAD")}

	bs, err := orig.Encode()
	ok(err)

	t.Logf("%#v turned into %d bytes", orig, len(bs))

	wm := &wendy.Message{Value: bs, Purpose: WendyPurpose,
		Key: addrToWendyID(to)}

	frw, err := msgFromWendy(wm)
	if err != nil {
		t.Fatalf("Couldn't turn Wendy msg %#v into a Message: %s", wm, err)
	}
	if !reflect.DeepEqual(orig, frw) {
		t.Fatalf("orig %#v != %#v", orig, frw)
	}
}

func TestAppDeliver(t *testing.T) {
	numMsgs := 10

	from, err := NewAddress([]byte("1111111111111111"))
	ok(err)
	to, err := NewAddress([]byte("2222222222222222"))
	ok(err)
	var sent []*Message

	app := newWendyApp(from)

	for i := range iter.N(numMsgs) {
		orig := &Message{Version: 23,
			From: from,
			To:   to,
			Data: []byte("DATS SUM PAYLOAD " + strconv.Itoa(i))}
		bs, err := orig.Encode()
		ok(err)

		wm := &wendy.Message{Purpose: WendyPurpose, Key: addrToWendyID(to),
			Value: bs}
		sent = append(sent, orig)
		app.OnDeliver(*wm)
	}

	var rcvd []*Message

	var wg sync.WaitGroup

	wg.Add(numMsgs)

	go func() {
		for msg := range app.delivd {
			rcvd = append(rcvd, msg)
			wg.Done()
		}
	}()

	wg.Wait()

	if len(rcvd) != numMsgs {
		t.Errorf("Expected %d messages, got %d", numMsgs, len(rcvd))
	}

	for i, rcv := range rcvd {
		if !reflect.DeepEqual(sent[i], rcv) {
			t.Errorf("sent[%d] %#v != %#v", i, sent[i], rcv)
		}
	}
}

func ExampleNodeReadFromI2aS() {
	payload := "hur de dur"
	tcpAddr1, _ := net.ResolveTCPAddr("tcp", "127.0.0.1:63300")
	i2addr1, _ := NewAddress([]byte("1111111111111111"))
	conf1 := &Config{LocalAddr: tcpAddr1, ExternalAddr: tcpAddr1, Address: i2addr1}

	got := make(chan *Message, 1)

	go func() {
		// initialize first node and start waiting for a message
		n, _ := conf1.Initialize()
		msg, _ := n.ReadFromI2aS()
		got <- msg
	}()

	tcpAddr2, _ := net.ResolveTCPAddr("tcp", "127.0.0.1:63301")
	i2addr2, _ := NewAddress([]byte("2222222222222222"))
	conf2 := &Config{LocalAddr: tcpAddr2, ExternalAddr: tcpAddr2, Address: i2addr2, BootstrapNode: tcpAddr1}

	<-time.After(1 * time.Second) // wait a bit so 1st node has had time to init

	n2, _ := conf2.Initialize() // bootstrap using 1st node

	n2.WriteToI2aS(&Message{TTL: 1, Data: []byte(payload)}, i2addr1) // write message to 1st node

	msg := <-got // wait for response
	fmt.Printf("%s -> %s: %s\n", msg.From, msg.To, string(msg.Data))
	// Output: 32323232323232323232323232323232 -> 31313131313131313131313131313131: hur de dur
}
