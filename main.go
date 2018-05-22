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
	buffer_size = 1024
)

func main() {
	log.Print("Port Forwarder")

	if len(os.Args) < 2 {
		log.Print("Incorrect parameters: needs <listen_address> and <target_address>")
		os.Exit(1)
	}

	listener_addr, err := net.ResolveTCPAddr("tcp", os.Args[1])
	if err != nil {
		log.Printf("Error resolving listen %s: %v", os.Args[1], err)
		os.Exit(1)
	}

	target_addr, err := net.ResolveTCPAddr("tcp", os.Args[2])
	if err != nil {
		log.Printf("Error resolving target %s: %v", os.Args[2], err)
		os.Exit(1)
	}

	listener, err := net.ListenTCP("tcp", listener_addr)
	if err != nil {
		log.Printf("Error listening: %v", err)
		os.Exit(1)
	}

	defer listener.Close()

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)

	go func() {
		for s := range signalChan {
			log.Printf("SIGNAL: %v", s)
			listener.Close()
		}
	}()

	connection_count := 0

	log.Print("Listening...")
	for {
		conn, err := listener.AcceptTCP()
		if err != nil {
			log.Printf("Error accepting: %v", err)
			os.Exit(2)
		}

		log.Printf("Client connected from (remote)%v", conn.RemoteAddr())

		connection_count += 1

		go forwardIt(connection_count, conn, target_addr)
	}

}

func forwardIt(conn_id int, conn *net.TCPConn, target_addr *net.TCPAddr) {
	defer conn.Close()

	remote_conn, err := net.DialTCP("tcp", nil, target_addr)
	if err != nil {
		log.Printf("forwardIt(%d) Error connecting to target: %v", conn_id, err)
		return
	}

	defer remote_conn.Close()

	log.Printf("forwardIt(%d) Connected to target", conn_id)

	go forwarder(fmt.Sprintf("forwarder(%d) remote -> local", conn_id), remote_conn, conn)

	forwarder(fmt.Sprintf("forwarder(%d) local -> remote", conn_id), conn, remote_conn)

	log.Printf("forwardIt(%d) exits", conn_id)
}

func forwarder(logheader string, in, out *net.TCPConn) {
	buffer := make([]byte, 1024)
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
