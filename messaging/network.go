// Package messaging provides DHT-based connectionless network I/O similar to UDP.
//
// Delivery or the order of messages is not guaranteed.
//
//
package messaging

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"time"

	"github.com/ORBAT/proxinator/util"
	"github.com/ORBAT/wendy"
)

const (
	ProtocolVersion = 0
	WendyPurpose    = 16
	NetworkName     = "i2msg"
)

func init() {
	rand.Seed(time.Now().UnixNano())
	log.SetFlags(log.Lshortfile | log.Lmicroseconds)
}

// Address represents the address of an I2aS messaging node
type Address wendy.NodeID

func (a Address) String() string {
	return fmt.Sprintf("%016x%016x", a[0], a[1])
}

func (a Address) Network() string {
	return NetworkName
}

func NewAddress(bs []byte) (addr Address, err error) {
	a, err := wendy.NodeIDFromBytes(bs)
	if err == nil {
		addr = Address(a)
	}
	return
}

func (a Address) MarshalBinary() (bs []byte, err error) {
	bs = make([]byte, 16)
	binary.BigEndian.PutUint64(bs, a[0])
	binary.BigEndian.PutUint64(bs[8:], a[1])
	return
}

func (a *Address) UnmarshalBinary(bs []byte) error {
	a[0] = binary.BigEndian.Uint64(bs)
	a[1] = binary.BigEndian.Uint64(bs[8:])
	return nil
}

func addrToWendyID(a Address) wendy.NodeID {
	return wendy.NodeID(a)
}

func addrFromWendyID(nid wendy.NodeID) Address {
	return Address(nid)
}

/*
net

type Addr interface {
        Network() string // name of the network
        String() string  // string form of address
}
*/

// Config holds configuration for the messaging node
type Config struct {
	// LocalAddr is the address and port the node is reachable on within its region
	LocalAddr *net.TCPAddr
	// ExternalAddr is the external address and port of the node
	ExternalAddr *net.TCPAddr
	// Region is the node's region. Nodes within the same region will be heavily favored by the routing algorithm. Can be omitted.
	Region string
	// BootstrapNode is the address and port of a node already in the network. Can be omitted if the node is the first one.
	BootstrapNode *net.TCPAddr
	// Address is the I2aS address to use. This is needed even when only sending messages.
	Address Address
}

func (nd *Node) initWendy() error {
	wn := wendy.NewNode(wendy.NodeID(nd.conf.Address), nd.conf.LocalAddr.IP.String(), nd.conf.ExternalAddr.IP.String(), nd.conf.Region, nd.conf.LocalAddr.Port)
	wcl := wendy.NewCluster(wn, nil) // TODO(ORBAT): credentials
	app := newWendyApp(nd.conf.Address)
	wcl.RegisterCallback(app)
	wcl.SetLogger(log.New(os.Stderr, fmt.Sprintf("[cluster-%s] ", wcl.ID().String()), log.Lmicroseconds|log.Lshortfile))
	wcl.SetLogLevel(wendy.LogLevelWarn)
	// TODO(ORBAT): graceful shutdown on signals etc.
	go wcl.Listen()

	if nd.conf.BootstrapNode != nil {
		nd.log.Printf("Bootstrapping using %s", nd.conf.BootstrapNode.String())
		err := wcl.Join(nd.conf.BootstrapNode.IP.String(), nd.conf.BootstrapNode.Port)
		if err != nil {
			return err
		}
		<-time.After(1 * time.Second) // give node some time to bootstrap
		nd.log.Println("Bootstrapped")
	}
	nd.wnode = wn
	nd.wcluster = wcl
	nd.wapp = app
	return nil
}

func (c *Config) initNode() *Node {
	id := <-util.SequentialInts
	lg := log.New(os.Stderr, fmt.Sprintf("[Node-%d %s] ", id, c.Address.String()), log.Lmicroseconds|log.Lshortfile)
	return &Node{conf: c, log: lg, id: id}
}

// Initialize initializes a node using the Config
func (c *Config) Initialize() (nd *Node, err error) {
	nd = c.initNode()
	err = nd.initWendy()
	if err != nil {
		return
	}

	return
}

// A Node is an initialized node in the messaging network. It implements the net.PacketConn interface.
// If a node isn't the first one in the network, it should bootstrap using a known node.
type Node struct {
	wcluster dhtIface
	wnode    *wendy.Node
	conf     *Config
	wapp     *wendyApp
	log      *log.Logger
	id       int
}

type dhtIface interface {
	Send(wendy.Message) error
	NewMessage(byte, wendy.NodeID, []byte) wendy.Message
	Stop()
}

func (nd *Node) Close() error {
	nd.wcluster.Stop()
	return nil
}

// WriteToI2aS writes message m to I2aS address addr using node nd, returning bytes written or an error. n only includes raw message byte count; the DHT's message
// "envelope" is not counted. The To field of m will be overwritten with addr.
func (nd *Node) WriteToI2aS(m *Message, addr Address) (n int, err error) {
	cl := m.Clone()
	cl.From = nd.conf.Address
	cl.To = addr

	bytes, err := cl.Encode()
	if err != nil {
		return
	}
	err = nd.wcluster.Send(nd.wcluster.NewMessage(16, wendy.NodeID(addr), bytes))
	if err == nil {
		n = len(bytes)
	}
	return
}

// WriteToI2aS writes bytes b to I2aS address addr using node nod, returning bytes written or an error. n only includes raw message byte count; the DHT's message
// "envelope" is not counted.
func (nd *Node) WriteTo(b []byte, addr net.Addr) (n int, err error) {
	//remember to check addr.Network()
	return 0, errors.New("wip")
}

// WriteToI2aS writes bytes b to I2aS address addr using node nod, returning bytes written or an error. n only includes raw message byte count; the DHT's message
// "envelope" is not counted.
func (n *Node) ReadFrom(b []byte) (int, net.Addr, error) {
	return 0, nil, errors.New("wip")
}

// TODO(ORBAT): timeouts for Node's net.PacketConn stuff

func (n *Node) ReadFromI2aS() (m *Message, err error) {
	n.log.Println("Waiting for a message")
	select {
	case msg, ok := <-n.wapp.delivd:
		n.log.Printf("Got message (%t)", ok)
		if ok {
			m = msg
		} else {
			err = errors.New("wendyApp delivered message channel closed?!")
		}
	}
	return
}

func (nd *Node) msgAsWendy(m *Message) wendy.Message {
	return nd.wcluster.NewMessage(WendyPurpose, wendy.NodeID(m.To), m.Data)
}

/*
type PacketConn interface {
        // ReadFrom reads a packet from the connection,
        // copying the payload into b.  It returns the number of
        // bytes copied into b and the return address that
        // was on the packet.
        // ReadFrom can be made to time out and return
        // an error with Timeout() == true after a fixed time limit;
        // see SetDeadline and SetReadDeadline.
        ReadFrom(b []byte) (n int, addr Addr, err error)

        // WriteTo writes a packet with payload b to addr.
        // WriteTo can be made to time out and return
        // an error with Timeout() == true after a fixed time limit;
        // see SetDeadline and SetWriteDeadline.
        // On packet-oriented connections, write timeouts are rare.
        WriteTo(b []byte, addr Addr) (n int, err error)

        // Close closes the connection.
        // Any blocked ReadFrom or WriteTo operations will be unblocked and return errors.
        Close() error

        // LocalAddr returns the local network address.
        LocalAddr() Addr

        // SetDeadline sets the read and write deadlines associated
        // with the connection.
        SetDeadline(t time.Time) error

        // SetReadDeadline sets the deadline for future Read calls.
        // If the deadline is reached, Read will fail with a timeout
        // (see type Error) instead of blocking.
        // A zero value for t means Read will not time out.
        SetReadDeadline(t time.Time) error

        // SetWriteDeadline sets the deadline for future Write calls.
        // If the deadline is reached, Write will fail with a timeout
        // (see type Error) instead of blocking.
        // A zero value for t means Write will not time out.
        // Even if write times out, it may return n > 0, indicating that
        // some of the data was successfully written.
        SetWriteDeadline(t time.Time) error
}
*/

// A Message is an I2aS messaging network message
// TODO(ORBAT):make message immutable
type Message struct {
	Version uint8 // protocol version
	From    Address
	To      Address
	TTL     uint16 // maximum number of hops to allow
	Data    []byte // message data
}

type message struct {
	Version *uint8
	From    *Address
	To      *Address
	TTL     *uint16
	Data    *[]byte
}

func (m *Message) GobEncode() (bs []byte, err error) {
	var buf bytes.Buffer
	ex := m.export()
	err = gob.NewEncoder(&buf).Encode(ex)

	if err == nil {
		bs = buf.Bytes()
	}

	return
}

func (m *Message) Encode() ([]byte, error) {
	return m.GobEncode()
}

func (m *Message) GobDecode(bs []byte) error {
	ex := m.export()
	dec := gob.NewDecoder(bytes.NewBuffer(bs))
	err := dec.Decode(ex)
	return err
}

func (m *Message) Decode(bs []byte) error {
	return m.GobDecode(bs)
}

func (m *Message) export() *message {
	return &message{&m.Version, &m.From, &m.To, &m.TTL, &m.Data}
}

func (m *Message) Clone() *Message {
	return &Message{m.Version, m.From, m.To, m.TTL, m.Data[:]}
}

func randomWendyID() (id wendy.NodeID) {
	id[0] = uint64(uint64(rand.Uint32())<<32 | uint64(rand.Uint32()))
	id[1] = uint64(uint64(rand.Uint32())<<32 | uint64(rand.Uint32()))
	return
}

func msgFromWendy(wm *wendy.Message) (m *Message, err error) {
	m = &Message{}
	err = m.Decode(wm.Value)
	return
}

const IncomingBufferSize = 10

type wendyApp struct {
	id     int
	addr   Address
	delivd chan *Message
	log    *log.Logger
}

func newWendyApp(a Address) *wendyApp {
	id := <-util.SequentialInts
	return &wendyApp{id, a, make(chan *Message, IncomingBufferSize), log.New(os.Stderr, fmt.Sprintf("[wendyApp-%d %s] ", id, a), log.Lmicroseconds|log.Lshortfile)}
}

func (app *wendyApp) OnError(err error) {
	panic(err.Error())
}

func (app *wendyApp) OnDeliver(wm wendy.Message) {
	app.log.Println("Got message")
	msg, err := msgFromWendy(&wm)
	if err != nil {
		app.OnError(err)
		return
	}
	app.delivd <- msg
	app.log.Println("Message delivered")
}

func (app *wendyApp) OnForward(msg *wendy.Message, next wendy.NodeID) bool {
	app.log.Printf("fwd %s -> %s.", msg.Key, next)
	return true // return false if you don't want the message forwarded
}

func (app *wendyApp) OnNewLeaves(leaves []*wendy.Node) {
	app.log.Print("Leaf set changed: ", leaves)
}

func (app *wendyApp) OnNodeJoin(node wendy.Node) {
	app.log.Print("Node joined: ", node.ID)
}

func (app *wendyApp) OnNodeExit(node wendy.Node) {
	app.log.Print("Node left: ", node.ID)
}

func (app *wendyApp) OnHeartbeat(node wendy.Node) {
	app.log.Print("Received heartbeat from ", node.ID)
}
