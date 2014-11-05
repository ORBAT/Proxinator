package main

import (
	"log"
	"math/rand"
	"time"

	"github.com/ORBAT/wendy"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// Address represents the address of an I2AS messaging node
type Address struct{}

/*
net

type Addr interface {
        Network() string // name of the network
        String() string  // string form of address
}
*/

// Config holds configuration for the messaging node
type Config struct {
	// LocalIP is the IP address the node is reachable on within its region
	LocalIP string
	// ExternalIP is the external IP of the node. The node must be reachable on this IP from the outside
	ExternalIP string
	// Region is the node's region. Nodes with the same region will be heavily favored by the routing algorithm. Can be omitted.
	Region string
	// Port is the port the node should listen on
	Port int
	// BootstrapNode is the address and port of a node already in the cluster
	BootstrapNode string
	// Address is the i2as address to use
	Address Address
}

// Initialize initializes a node using the Config
func (c *Config) Initialize() (Node, error) {}

// A Node is an initialized node in the messaging network. If a node isn't the first one in the network, it should bootstrap using a known node
type Node struct{}

// Dial connects to the remote address raddr
func (n *Node) Dial(raddr *Address) (*Conn, error) {}

/*
type Conn interface {
        // Read reads data from the connection.
        // Read can be made to time out and return a Error with Timeout() == true
        // after a fixed time limit; see SetDeadline and SetReadDeadline.
        Read(b []byte) (n int, err error)

        // Write writes data to the connection.
        // Write can be made to time out and return a Error with Timeout() == true
        // after a fixed time limit; see SetDeadline and SetWriteDeadline.
        Write(b []byte) (n int, err error)

        // Close closes the connection.
        // Any blocked Read or Write operations will be unblocked and return errors.
        Close() error

        // LocalAddr returns the local network address.
        LocalAddr() Addr

        // RemoteAddr returns the remote network address.
        RemoteAddr() Addr

        // SetDeadline sets the read and write deadlines associated
        // with the connection. It is equivalent to calling both
        // SetReadDeadline and SetWriteDeadline.
        //
        // A deadline is an absolute time after which I/O operations
        // fail with a timeout (see type Error) instead of
        // blocking. The deadline applies to all future I/O, not just
        // the immediately following call to Read or Write.
        //
        // An idle timeout can be implemented by repeatedly extending
        // the deadline after successful Read or Write calls.
        //
        // A zero value for t means I/O operations will not time out.
        SetDeadline(t time.Time) error

        // SetReadDeadline sets the deadline for future Read calls.
        // A zero value for t means Read will not time out.
        SetReadDeadline(t time.Time) error

        // SetWriteDeadline sets the deadline for future Write calls.
        // Even if write times out, it may return n > 0, indicating that
        // some of the data was successfully written.
        // A zero value for t means Write will not time out.
        SetWriteDeadline(t time.Time) error
}
*/

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

// A Conn is the implementation of the Conn and PacketConn interfaces for the I2AS messaging network
type Conn struct{}

/*
type Listener interface {
        // Accept waits for and returns the next connection to the listener.
        Accept() (c Conn, err error)

        // Close closes the listener.
        // Any blocked Accept operations will be unblocked and return errors.
        Close() error

        // Addr returns the listener's network address.
        Addr() Addr
}
*/

// A Listener is a I2AS messaging network listener
type Listener struct{}

type MessageType uint8

// A Message is an I2AS messaging network message
type Message struct {
	Version uint16
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
