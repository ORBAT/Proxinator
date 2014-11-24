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

// Messages sent on this channel will be readable using Node's ReadFromI2aS or ReadFrom
func (mc *mockCluster) msgDelivery() chan *Message {
	return mc.app.(*wendyApp).delivd
}

func newMockConf(a Address) *Config {
	return &Config{Address: a}
}

const msgBufSize = 20

func newMockNode(a Address) (*Node, *mockCluster, <-chan *wendy.Message) {
	conf := newMockConf(a)
	nd := conf.initNode()

	sentCh := make(chan *wendy.Message, msgBufSize)
	app := newWendyApp(a)
	mc := newMockCluster(sentCh, app)

	nd.wcluster = mc
	nd.wapp = app
	return nd, mc, sentCh
}

func ok(v interface{}) {
	if v != nil {
		panic(v)
	}
}

func TestWriteTo(t *testing.T) {
	from := Address(randomWendyID())
	to := Address(randomWendyID())

	nd, _, sentCh := newMockNode(from)
	bytes := []byte{1, 75, 31, 44, 8, 1, 100, 20}

	t.Logf("From %s\nTo %s\n", from, to)

	n, err := nd.WriteTo(bytes, to)
	if err != nil {
		t.Fatal("WriteTo error:", err)
	}
	if n == 0 {
		t.Error("WriteTo gave n = 0?")
	}

	select {
	case sent := <-sentCh:
		recvmsg := new(Message)
		err := recvmsg.Decode(sent.Value)
		ok(err)
		t.Logf("recv\n%#v", recvmsg)
		if recvmsg.From != from {
			t.Errorf("Wanted From %s, got %s", from, recvmsg.From)
		}

		if recvmsg.To != to {
			t.Errorf("Wanted To %s, got %s", to, recvmsg.To)
		}
	case <-time.After(50 * time.Millisecond):
		t.Fatal("Message wasn't sent?")
	}
}

func TestWriteToI2aS(t *testing.T) {
	from := Address(randomWendyID())
	to := Address(randomWendyID())

	nd, _, sentCh := newMockNode(from)
	msg := &Message{Version: ProtocolVersion, Data: []byte{1, 1, 2, 3, 5, 8}}

	t.Logf("From %s\nTo %s\n", from, to)

	n, err := nd.WriteToI2aS(msg, to)
	if err != nil {
		t.Fatal("WriteToI2aS error:", err)
	}
	if n == 0 {
		t.Error("WriteToI2aS gave n = 0?")
	}
	select {
	case sent := <-sentCh:
		recvmsg := new(Message)
		err := recvmsg.Decode(sent.Value)
		ok(err)
		t.Logf("recv\n%#v", recvmsg)
		if recvmsg.From != from {
			t.Errorf("Wanted From %s, got %s", from, recvmsg.From)
		}

		if recvmsg.To != to {
			t.Errorf("Wanted To %s, got %s", to, recvmsg.To)
		}

		if !reflect.DeepEqual(recvmsg.Data, msg.Data) {
			t.Errorf("Wanted Data %#v, got %#v", recvmsg.Data, msg.Data)
		}
	case <-time.After(50 * time.Millisecond):
		t.Fatal("Message wasn't sent?")
	}
}

func TestReadFromI2aS(t *testing.T) {
	from := Address(randomWendyID())
	to := Address(randomWendyID())

	nd, mc, _ := newMockNode(to)

	numMsgs := msgBufSize - 1

	retCh := make(chan *Message, numMsgs)

	reader := func(ret chan *Message) {
		for {
			msg, err := nd.ReadFromI2aS()
			ret <- msg
			if err != nil {
				close(ret)
				t.Error("ReadFromI2aS error:", err.Error())
			}
		}
	}

	go reader(retCh)

	var wg sync.WaitGroup
	wg.Add(numMsgs)
	var msgs []*Message

	for i := range iter.N(numMsgs) {
		msg := &Message{Version: ProtocolVersion, From: from, To: to, Data: []byte{byte(i % 256)}}
		msgs = append(msgs, msg)
		go func(msg *Message, w *sync.WaitGroup) {
			defer w.Done()
			mc.msgDelivery() <- msg
		}(msg, &wg)
	}

	wg.Wait()

	nrcvd := 0
	var rcvd []*Message

	for {
		select {
		case <-time.After(50 * time.Millisecond):
			goto Out
		case msg := <-retCh:
			rcvd = append(rcvd, msg)
			nrcvd++
		}
	}

Out:

	if nrcvd != numMsgs {
		t.Errorf("Expected %d messages, got %d", numMsgs, nrcvd)
	}

	for i, msg := range rcvd {
		if msg.Data[0] != byte(i%256) {
			t.Errorf("Odd message at idx %d (%d), %#v", i, i%256, msg)
		}
		if msg.From != from {
			t.Errorf("Wanted From %s, got %s", msg.From, from)
		}
		if msg.To != to {
			t.Errorf("Wanted to %s, got %s", msg.To, to)
		}
	}

}

func TestReadFrom(t *testing.T) {
	from := Address(randomWendyID())
	to := Address(randomWendyID())

	nd, mc, _ := newMockNode(to)

	numMsgs := msgBufSize - 1

	retCh := make(chan []byte, numMsgs)

	reader := func(ret chan []byte) {
		for {
			buf := make([]byte, 32)
			n, addr, err := nd.ReadFrom(buf)
			buf = buf[:n]
			if err != nil {
				t.Error("ReadFrom error:", err.Error())
			}
			if addr != from {
				t.Errorf("Expected message from %s but got one from %s", from, addr)
			}
			ret <- buf
		}
	}

	go reader(retCh)

	var wg sync.WaitGroup
	wg.Add(numMsgs)
	var msgs [][]byte

	for i := range iter.N(numMsgs) {
		msg := []byte{byte(i % 256)}
		msgs = append(msgs, msg)
		go func(bs []byte, w *sync.WaitGroup) {
			defer w.Done()
			mc.msgDelivery() <- &Message{From: from, Data: bs}
		}(msg, &wg)
	}

	wg.Wait()

	nrcvd := 0
	var rcvd [][]byte

	for {
		select {
		case <-time.After(50 * time.Millisecond):
			goto Out
		case msg, ok := <-retCh:
			if ok {
				rcvd = append(rcvd, msg)
				nrcvd++
			} else {
				t.Error("retCh closed")
			}

		}
	}

Out:

	if nrcvd != numMsgs {
		t.Errorf("Expected %d messages, got %d", numMsgs, nrcvd)
	}

	for i, msg := range rcvd {
		if len(msg) != 1 {
			t.Error("Weird messge length", len(msg))
			continue
		}
		if msg[0] != byte(i%256) {
			t.Errorf("Odd message at idx %d (%d), %#v", i, i%256, msg)
		}
	}

}

func TestAddressEncoding(t *testing.T) {
	orig := Address(randomWendyID())
	bytes, err := orig.MarshalBinary()
	if err != nil {
		t.Fatal("Couldn't marshal address:", err)
	}

	unm := new(Address)
	err = unm.UnmarshalBinary(bytes)
	t.Logf("bytes %#v, unm %#v", bytes, unm)
	if err != nil {
		t.Fatal("Couldn't unmarshal address:", err)
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

func TestAppForward(t *testing.T) {

}

func TestAppDeliver(t *testing.T) {
	// defer logging.LogTo(os.Stderr)()
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

	ok(app.Stop())

	var rcvd []*Message

	var wait sync.WaitGroup
	wait.Add(1)

	go func() {
		defer wait.Done()
		for msg := range app.delivd {
			rcvd = append(rcvd, msg)
		}
	}()

	wait.Wait()

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
	// Outputs: 32323232323232323232323232323232 -> 31313131313131313131313131313131: hur de dur
}
