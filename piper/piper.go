// package piper proxies network connections through another host
package piper

import (
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"log"
	"net"

	"github.com/ORBAT/proxinator/logging"
	"github.com/petar/GoTeleport/tele"
	"github.com/petar/GoTeleport/tele/blend"
)

var (
	ErrInvalidMsgType = errors.New("Invalid message type")
)

const (
	ReadBufSize = 64 * 1024 // size of read buffer used by copy
)

var proxyLog = logging.New("ConnProxy", nil)

type dataMsg struct {
	Data []byte
}

func init() {
	gob.Register(&dialMsg{})
	gob.Register(&dataMsg{})
}

type muxConn blend.Conn

// Read reads byte slices from a multiplexed connection. It only receives messsage
// type dataMsg and will return ErrInvalidMstType if it encounters anything else.
func (mc *muxConn) Read(p []byte) (n int, err error) {
	r, err := (*blend.Conn)(mc).Read()
	if err != nil {
		return
	}
	dmsg, ok := r.(*dataMsg)
	if !ok {
		return 0, ErrInvalidMsgType
	}

	n = copy(p, dmsg.Data)
	if n == 0 {
		err = io.ErrNoProgress
	}
	return
}

// dialMsg contains the network and address a node wants to connect to.
// MUST be the first message sent on a connection.
type dialMsg struct {
	Network string
	Address string
}

func NewNode(muxAddr net.Addr, inListener net.Listener) (*Node, error) {
	log := logging.New("mux_node@%s", nil)
	t := tele.NewStructOverTCP()
	l := t.Listen(muxAddr)
	_ = &Node{muxAddr: muxAddr, inListener: inListener, muxListener: l, log: log}
	return nil, errors.New("NewNode WIP")
}

type Node struct {
	muxAddr     net.Addr        // the multiplexer address of this node. Other nodes connect to this
	inListener  net.Listener    // listener for incoming "normal" connections
	muxListener *blend.Listener // listener for muxed connections
	log         *log.Logger
}

// DialProxied dials address through
func (nd *Node) DialProxied(net, addr string, through net.Addr) (*ProxiedConn, error) {
	return nil, errors.New("Dial WIP")
}

func (nd *Node) muxListenLoop() {
	for {
		conn := nd.muxListener.Accept()
		rawMsg, err := conn.Read()
		ok(err)
		msg, succ := rawMsg.(*dialMsg)
		if !succ {
			panic(fmt.Errorf("%#v not a *dialMsg?", msg))
		}
		_, err = net.Dial(msg.Network, msg.Address)
		ok(err)

	}
}

// A ProxiedConn is a network connection proxied through another host
type ProxiedConn struct {
}

func ProxyThrough(to net.Addr) (error, *ProxiedConn) {
	return errors.New("ProxyThrough WIP"), nil
}

// Copy copies from src to dst until either EOF is reached
// on src or an error occurs, and in both cases dst will be closed.
// It returns the number of bytes copied and the first error encountered
// while copying, if any.
//
// A successful Copy returns err == nil, not err == EOF.
// Because Copy is defined to read from src until EOF, it does
// not treat an EOF from Read as an error to be reported.
func Copy(dst io.WriteCloser, src io.Reader) (written int64, err error) {
	buf := make([]byte, ReadBufSize)
	for {
		nr, er := src.Read(buf)
		if nr > 0 {
			nw, ew := dst.Write(buf[0:nr])
			if nw > 0 {
				written += int64(nw)
			}
			if ew != nil {
				err = ew
				break
			}
			if nr != nw {
				err = io.ErrShortWrite
				break
			}
		}
		if er == io.EOF {
			break
		}
		if er != nil {
			err = er
			break
		}
	}

	if ec := dst.Close(); ec != nil {
		err = ec
	}

	return written, err
}

func ok(v interface{}) {
	if v != nil {
		panic(v)
	}
}
