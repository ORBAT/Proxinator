package main

import (
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/signal"

	"strconv"
	"syscall"
	"time"

	flags "github.com/jessevdk/go-flags"

	"github.com/ORBAT/wendy"
)

var options MessagingConfig

var fp = flags.NewParser(&options, flags.Default)

func main() {
	_, err := fp.Parse()
	if err != nil {
		os.Exit(1)
	}

	if len(options.IDString) == 0 {
		options.nodeID = randomID()
		options.IDString = options.nodeID.String()
	} else {
		options.nodeID, err = wendy.NodeIDFromBytes([]byte(options.IDString))
		if err != nil {
			panic(err)
		}
	}

	options.v = len(options.Verbose)

	node := wendy.NewNode(options.nodeID, options.LocalIP, options.ExternalIP, options.Region, options.Port)
	log.Printf("Node ID %s (%s:%d / %s:%d) %d region %s, bootstrap %s", options.nodeID, options.LocalIP, options.Port, options.ExternalIP, options.Port, node.Port, options.Region, options.BootstrapNode)
	cluster := wendy.NewCluster(node, nil)

	cluster.SetHeartbeatFrequency(10)
	cluster.SetNetworkTimeout(1)
	cluster.SetLogLevel(wendy.LogLevelWarn)

	if options.v > 0 {
		log.SetOutput(os.Stderr)
		log.Printf("Verbosity %d", options.v)
		if options.v > 1 {
			cluster.SetLogLevel(wendy.LogLevelDebug)
		}
	} else {
		log.SetOutput(ioutil.Discard)
	}

	sigCh := make(chan os.Signal)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	cluster.RegisterCallback(&debugWendy{})
	go func() {
		err := cluster.Listen()
		if err != nil {
			panic(err)
		}
	}()

	if len(options.BootstrapNode) > 0 {
		if host, portStr, err := net.SplitHostPort(options.BootstrapNode); err != nil {
			panic(err)
		} else {
			port, err := strconv.Atoi(portStr)
			if err != nil {
				panic(err)
			}
			<-time.After(2 * time.Second)
			log.Printf("Bootstrapping using %s:%d", host, port)
			err = cluster.Join(host, port)
			if err != nil {
				panic(err)
			}
		}
	}

	select {
	case sig := <-sigCh:
		log.Print(sig)
		cluster.Stop()
		<-time.After(1 * time.Second)
	}

}
