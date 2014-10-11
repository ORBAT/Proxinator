package main

import (
	"log"
	"math/rand"
	"time"

	"github.com/ORBAT/wendy"
)

// MessagingConfig holds configuration for the messaging component
type MessagingConfig struct {
	// LocalIP is the IP address the node is reachable on within its region
	LocalIP string `short:"i" long:"localip" default:"127.0.0.1" description:"LocalIP is the IP address the node is reachable on within its region"`
	// ExternalIP is the external IP of the node. The node is globally reachable on this IP
	ExternalIP string `short:"e" long:"externalip" default:"127.0.0.1" description:"ExternalIP is the external IP of the node. The node is globally reachable on this IP"`
	// Region is the node's region. Nodes with the same region will be heavily favored by the routing algorithm. Can be omitted.
	Region string `short:"r" long:"region" description:"Region is the node's region. Nodes with the same region will be heavily favored by the routing algorithm. Can be omitted."`
	// Port is the port the messaging componen should listen on
	Port int `short:"p" long:"port" default:"39000" description:"Port is the port the messaging componen should listen on"`
	// ID is the node ID. Must have at least 16 characters (anything over 16 is trimmed). If omitted, a random ID is used
	IDString string `short:"I" long:"id" default:"" description:"ID is the node ID. Must have at least 16 characters (anything over 16 is trimmed). If omitted, a random ID is used"`
	// BootstrapNode is the address and port of a node already in the cluster
	BootstrapNode string `short:"b" long:"bootstrap" description:"BootstrapNode is the address and port of a node already in the cluster. Can be omitted"`
	nodeID        wendy.NodeID
	Verbose       []bool `short:"v" long:"verbose" description:"Verbosity. Can be given multiple times"`
	v             int    // verbosity level as a number
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

func randomID() (id wendy.NodeID) {
	id[0] = uint64(uint64(rand.Uint32())<<32 | uint64(rand.Uint32()))
	id[1] = uint64(uint64(rand.Uint32())<<32 | uint64(rand.Uint32()))
	return
}

type debugWendy struct {
}

func (app *debugWendy) OnError(err error) {
	panic(err.Error())
}

func (app *debugWendy) OnDeliver(msg wendy.Message) {
	log.Print("Received message: ", msg)
}

func (app *debugWendy) OnForward(msg *wendy.Message, next wendy.NodeID) bool {
	log.Printf("Forwarding message %s to Node %s.", msg.Key, next)
	return true // return false if you don't want the message forwarded
}

func (app *debugWendy) OnNewLeaves(leaves []*wendy.Node) {
	log.Print("Leaf set changed: ", leaves)
}

func (app *debugWendy) OnNodeJoin(node wendy.Node) {
	log.Print("Node joined: ", node.ID)
}

func (app *debugWendy) OnNodeExit(node wendy.Node) {
	log.Print("Node left: ", node.ID)
}

func (app *debugWendy) OnHeartbeat(node wendy.Node) {
	log.Print("Received heartbeat from ", node.ID)
}
