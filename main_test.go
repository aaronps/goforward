package main

import (
	"net"
	"os"
	"testing"
	"time"
)

func interruptMain() {
	signalChan <- os.Interrupt
}

func TestLessArguments(t *testing.T) {
	if cmain([]string{"main", "tcp", "sss"}) != exitBadArguments {
		t.Error("Incorrect exit value for less parameters")
	}
}

func TestInvalidModeArgument(t *testing.T) {
	if cmain([]string{"main", "invalid_mode", "1", "2"}) != exitBadArguments {
		t.Error("Incorrect exit value for less parameters")
	}
}

func TestInvalidListenAddr(t *testing.T) {
	if cmain([]string{"main", "tcp", "1:2.3", "2"}) != exitResolve {
		t.Error("Should return resolving error")
	}
}

func TestInvalidTargetAddress(t *testing.T) {
	if cmain([]string{"main", "tcp", "localhost:0", "1:2.3"}) != exitResolve {
		t.Error("Should return resolving error")
	}
}

func TestForwardSingleUDP(t *testing.T) {
	udpReceiver, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Log("Cannot prepare udp receiver", err)
		t.FailNow()
	}

	defer udpReceiver.Close()

	udpReceiverAddr := udpReceiver.LocalAddr().(*net.UDPAddr)

	udpListenerAddr, _ := net.ResolveUDPAddr("udp", "127.0.0.1:29990")

	receiveChan := make(chan bool, 1)
	resultChan := make(chan bool, 1)
	sendBuffer := [8]byte{1, 2, 3, 4, 5, 6, 7, 8}

	go func() {
		resultChan <- listenAndForwardUDP(udpListenerAddr, []*net.UDPAddr{udpReceiverAddr}) == exitOk
	}()

	go func() {
		udpReceiver.SetDeadline(time.Now().Add(200 * time.Millisecond))
		var rbuffer [8]byte

		rcount, _, _ := udpReceiver.ReadFrom(rbuffer[:])
		if rcount != len(sendBuffer) {
			receiveChan <- false
		} else {
			receiveChan <- rbuffer == sendBuffer
		}
	}()

	time.Sleep(100 * time.Millisecond)

	udpSender, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Log("Cannot prepare udp sender", err)
		t.FailNow()
	}

	defer udpSender.Close()

	udpSender.(*net.UDPConn).WriteToUDP(sendBuffer[:], udpListenerAddr)

	if !<-receiveChan {
		t.Error("Receiver didn't get it")
	}

	interruptMain()
}

func TestForwardSingleTCP(t *testing.T) {

	tcpReceiver, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Log("Unable to listen", err)
		t.FailNow()
	}

	defer tcpReceiver.Close()

	tcpReceiverAddr := tcpReceiver.Addr().(*net.TCPAddr)

	tcpListenerAddr, _ := net.ResolveTCPAddr("tcp", "127.0.0.1:29990")

	receiveChan := make(chan bool, 1)
	resultChan := make(chan bool, 1)
	sendBuffer := [8]byte{1, 2, 3, 4, 5, 6, 7, 8}

	go func() {
		resultChan <- listenAndForwardTCP(tcpListenerAddr, []*net.TCPAddr{tcpReceiverAddr}) == exitOk
	}()

	go func() {
		conn, _ := tcpReceiver.Accept()
		defer conn.Close()

		conn.SetDeadline(time.Now().Add(200 * time.Millisecond))
		var rbuffer [8]byte

		rcount, _ := conn.Read(rbuffer[:])
		if rcount != len(sendBuffer) {
			receiveChan <- false
		} else {
			receiveChan <- rbuffer == sendBuffer
		}
	}()

	// wait a little in hope the listener is ready
	time.Sleep(100 * time.Millisecond)

	sendConn, _ := net.Dial("tcp", "127.0.0.1:29990")
	defer sendConn.Close()

	sendConn.Write(sendBuffer[:])

	if !<-receiveChan {
		t.Error("Receiver didn't get it")
	}

	interruptMain()
}
