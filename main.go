// main
package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
)

const (
	tcpBufferSize = 1492
	udpBufferSize = 65536
	appVersion    = "1.0.0"
)

// This part is to ensure only the defined exit codes are used
type exitCode interface {
	exitValue() int
}

// exitType should be keep secret
type exitType int

func (r exitType) exitValue() int {
	return int(r)
}

// the available exit codes
const (
	exitOk exitType = iota
	exitBadArguments
	exitResolve
	exitListen
	exitRead
)

var (
	// created here so we can use it on tests
	signalChan  = make(chan os.Signal, 1)
	interrupted = false
)

func main() {
	os.Exit(cmain(os.Args).exitValue())
}

func cmain(args []string) exitCode {
	log.Printf("Port Forwarder V. %s", appVersion)

	if len(args) < 4 {
		printUsage(args[0])
		return exitBadArguments
	}

	numAddresses := len(os.Args) - 2

	switch args[1] {
	case "tcp":
		resolvedAddresses := make([]*net.TCPAddr, numAddresses)

		for index, host := range args[2:] {
			addr, err := net.ResolveTCPAddr("tcp", host)
			if err != nil {
				printResolveError(index == 0, host, err)
				return exitResolve
			}

			resolvedAddresses[index] = addr
		}

		return listenAndForwardTCP(resolvedAddresses[0], resolvedAddresses[1:])

	case "udp":
		resolvedAddresses := make([]*net.UDPAddr, numAddresses)

		for index, host := range args[2:] {
			addr, err := net.ResolveUDPAddr("udp", host)
			if err != nil {
				printResolveError(index == 0, host, err)
				return exitResolve
			}

			resolvedAddresses[index] = addr
		}

		return listenAndForwardUDP(resolvedAddresses[0], resolvedAddresses[1:])
	}

	printUsage(args[0])
	return exitBadArguments
}

func printUsage(cmdname string) {
	log.Printf("Usage: %s <tcp|udp> <listen_address> <target_address1> [...<target_addressN>]", cmdname)
}

func printResolveError(isListen bool, host string, err error) {
	if isListen {
		log.Printf("Error resolving listen address %s: %v", host, err)
	} else {
		log.Printf("Error resolving address %s: %v", host, err)
	}
}

type closeable interface {
	Close() error
}

func closeOnSignal(conn closeable) {
	interrupted = false
	signal.Notify(signalChan, os.Interrupt)

	go func() {
		for s := range signalChan {
			log.Printf("SIGNAL: %v", s)
			interrupted = true
			conn.Close()
			break
		}
	}()
}

func listenAndForwardUDP(listenAddr *net.UDPAddr, targetList []*net.UDPAddr) exitCode {
	listener, err := net.ListenUDP("udp", listenAddr)
	if err != nil {
		log.Printf("Error listening: %v", err)
		return exitListen
	}

	defer listener.Close()
	closeOnSignal(listener)

	buffer := make([]byte, udpBufferSize)
	targetIndex := 0
	targetCount := len(targetList)

	for {
		size, err := listener.Read(buffer)
		if err != nil {
			if interrupted {
				log.Printf("Interrupted, bye")
				return exitOk
			}

			log.Printf("Error reading UDP: %v", err)
			return exitRead
		}

		log.Printf("Forward UDP packet of size %v to %v", size, targetList[targetIndex])

		listener.WriteToUDP(buffer[:size], targetList[targetIndex])

		if targetCount > 1 {
			targetIndex++
			if targetIndex > targetCount {
				targetIndex = 0
			}
		}
	}
}

func listenAndForwardTCP(listenAddr *net.TCPAddr, targetList []*net.TCPAddr) exitCode {
	listener, err := net.ListenTCP("tcp", listenAddr)
	if err != nil {
		log.Printf("Error listening: %v", err)
		return exitListen
	}

	defer listener.Close()
	closeOnSignal(listener)

	connectionCount := 0
	targetIndex := 0
	targetCount := len(targetList)

	log.Printf("Listening on %s", listenAddr.String())
	for {
		conn, err := listener.AcceptTCP()
		if err != nil {
			if interrupted {
				log.Printf("Interrupted, bye")
				return exitOk
			}

			log.Printf("Error accepting: %v", err)
			return exitRead
		}

		log.Printf("Client connected from (remote)%v", conn.RemoteAddr())

		connectionCount++

		go forwardTCP(connectionCount, conn, targetList[targetIndex])

		if targetCount > 1 {
			targetIndex++
			if targetIndex > targetCount {
				targetIndex = 0
			}
		}
	}
}

func forwardTCP(connID int, conn *net.TCPConn, targetAddr *net.TCPAddr) {
	defer conn.Close()

	remoteConn, err := net.DialTCP("tcp", nil, targetAddr)
	if err != nil {
		log.Printf("forwardIt(%d) Error connecting to target: %v", connID, err)
		return
	}

	defer remoteConn.Close()

	log.Printf("forwardIt(%d) Connected to target", connID)

	go forwarder(fmt.Sprintf("forwarder(%d) remote -> local", connID), remoteConn, conn)

	forwarder(fmt.Sprintf("forwarder(%d) local -> remote", connID), conn, remoteConn)

	log.Printf("forwardIt(%d) exits", connID)
}

func forwarder(logheader string, in, out *net.TCPConn) {
	buffer := make([]byte, tcpBufferSize)
	for {
		readed, err := in.Read(buffer)
		if err == io.EOF {
			log.Printf("%s disconnected", logheader)
			break
		}
		if err != nil {
			log.Printf("%s read error: %v", logheader, err)
			break
		}

		wrote, err := out.Write(buffer[:readed])
		if err != nil {
			log.Printf("%s write error: %v", logheader, err)
			break
		}

		if wrote != readed {
			log.Printf("%s write error (didn't write everything) %d of %d", logheader, wrote, readed)
			break
		}
	}
}
