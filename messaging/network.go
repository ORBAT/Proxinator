// Package messaging provides DHT-based connectionless network I/O similar to UDP.
//
// Delivery or the order of messages is not guaranteed.
//
//
package messaging

import (
	"log"
	"math/rand"
	"net"
	"time"

	"github.com/ORBAT/wendy"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// Address represents the address of an I2aS messaging node
type Address []byte

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
	LocalAddr net.TCPAddr
	// ExternalAddr is the external address and port of the node
	ExternalAddr net.TCPAddr
	// Region is the node's region. Nodes within the same region will be heavily favored by the routing algorithm. Can be omitted.
	Region string
	// BootstrapNode is the address and port of a node already in the network. Can be omitted if the node is the first one.
	BootstrapNode net.TCPAddr
	// Address is the I2aS address to use. This is needed even when only sending messages.
	Address Address
}

// Initialize initializes a node using the Config
func (c *Config) Initialize() (Node, error) {}

// A Node is an initialized node in the messaging network. It implements the net.PacketConn interface.
// If a node isn't the first one in the network, it should bootstrap using a known node.
type Node struct {
	wcluster *wendy.Cluster
	wnode    *wendy.Node
	conf     *Config
}

func (n *Node) WriteToI2aS(m *Message, addr Address) error {}

func (n *Node) WriteTo(b []byte, addr net.Addr) (int, error) {}

func (n *Node) ReadFrom(b []byte) (int, net.Addr, error) {}

func (n *Node) ReadFromI2aS() (Message, error) {}

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
type Message struct {
	Version uint8 // protocol version
	From    Address
	To      Address
	Data    []byte // message data
}

func randomWendyID() (id wendy.NodeID) {
	id[0] = uint64(uint64(rand.Uint32())<<32 | uint64(rand.Uint32()))
	id[1] = uint64(uint64(rand.Uint32())<<32 | uint64(rand.Uint32()))
	return
}

type wendyApp struct{}

func (app *wendyApp) OnError(err error) {
	panic(err.Error())
}

func (app *wendyApp) OnDeliver(msg wendy.Message) {
	log.Print("Received message: ", msg)
}

func (app *wendyApp) OnForward(msg *wendy.Message, next wendy.NodeID) bool {
	log.Printf("Forwarding message %s to Node %s.", msg.Key, next)
	return true // return false if you don't want the message forwarded
}

func (app *wendyApp) OnNewLeaves(leaves []*wendy.Node) {
	log.Print("Leaf set changed: ", leaves)
}

func (app *wendyApp) OnNodeJoin(node wendy.Node) {
	log.Print("Node joined: ", node.ID)
}

func (app *wendyApp) OnNodeExit(node wendy.Node) {
	log.Print("Node left: ", node.ID)
}

func (app *wendyApp) OnHeartbeat(node wendy.Node) {
	log.Print("Received heartbeat from ", node.ID)
}
