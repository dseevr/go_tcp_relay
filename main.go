package main

import (
	"bufio"
	"container/list"
	"flag"
	"log"
	"net"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

var clientMutex sync.Mutex
var clients list.List // of net.Conn

var incomingClientsCount int64

var messagesReceived, messagesSent int64

var incomingAddress, outgoingAddress string

var messagesLastMinute int64
var messageRateMutex sync.Mutex

func statusReport() {
	clientMutex.Lock()
	log.Printf("Incoming clients connected: %v\n", incomingClientsCount)
	log.Printf("Outbound clients connected: %d\n", clients.Len())
	log.Printf("Messages received/sent: %d/%d\n", messagesReceived, messagesSent)
	clientMutex.Unlock()
}

func addConn(n net.Conn) *list.Element {
	clientMutex.Lock()
	l := clients.PushBack(n)
	clientMutex.Unlock()

	return l
}

func removeConn(l *list.Element) {
	clientMutex.Lock()
	clients.Remove(l)
	clientMutex.Unlock()
}

func sendMsg(msg string) {
	clientMutex.Lock()
	for e := clients.Front(); e != nil; e = e.Next() {
		_, err := e.Value.(net.Conn).Write([]byte(msg))
		if err != nil {
			panic(err)
		}

		messagesSent += 1
	}
	clientMutex.Unlock()
}

func sendKeepalives() {
	for _ = range time.Tick(1 * time.Minute) {
		// log.Println("Sending keepalive pings")
		sendMsg("PING\n")
	}
}

func listenForIncomingClients() {
	ln, err := net.Listen("tcp", incomingAddress)
	if err != nil {
		panic(err)
	}
	log.Println("Listening for incoming clients on", incomingAddress)

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println("ln.Accept:", err)
			continue
		}

		log.Println("Incoming client connected")

		incomingClientsCount += 1

		go func() {
			defer func() {
				if r := recover(); r != nil {
					log.Println("Incoming client disconnected:", r)
					atomic.AddInt64(&incomingClientsCount, -1)
				}
			}()

			// log.Println("Waiting for message")

			scanner := bufio.NewScanner(conn)
			for scanner.Scan() {
				msg := scanner.Text()

				if(msg == "PING") {
					log.Println("Received PING from incoming client")
					continue
				}

				// log.Printf("Received %d bytes: %s\n", len(msg), msg)

				atomic.AddInt64(&messagesReceived, 1)
				atomic.AddInt64(&messagesLastMinute, 1)

				sendMsg(msg + "\n")
			}

			if err := scanner.Err(); err != nil {
				log.Println("scanner.Scan:", err)
			}
		}()
	}
}

func listenForOutgoingClients() {
	ln, err := net.Listen("tcp", outgoingAddress)
	if err != nil {
		panic(err)
	}
	log.Println("Listening for outgoing clients on", outgoingAddress)

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println("ln.Accept:", err)
			continue
		}

		go func() {
			log.Println("Outgoing client connected")
			element := addConn(conn)

			defer func() {
				if r := recover(); r != nil {
					log.Println("Outgoing client disconnected:", r)
					removeConn(element)
				}
			}()

			for _ = range time.Tick(5 * time.Second) {
				// do nothing
			}
		}()
	}
}

func main() {
	runtime.GOMAXPROCS(16)

	messagesReceived = int64(0)
	messagesSent = int64(0)
	incomingClientsCount = int64(0)
	messagesLastMinute = int64(0)

	flag.StringVar(&incomingAddress, "incoming-address", "localhost:12345", "local host:port to receive data on")
	flag.StringVar(&outgoingAddress, "outgoing-address", "localhost:54321", "target host:port to send data on")
	flag.Parse()

	go listenForIncomingClients()
	go listenForOutgoingClients()

	go func() {
		for _ = range time.Tick(5 * time.Minute) {
			statusReport()
		}
	}()

	go func() {
		for _ = range time.Tick(1 * time.Minute) {
			messageRateMutex.Lock()
			perSecondAverage := float64(messagesLastMinute) / 60.0
			log.Printf("Messages last minute: %d (%.2f/s average)\n", messagesLastMinute, perSecondAverage)
			messagesLastMinute = 0
			messageRateMutex.Unlock()
		}
	}()

	go sendKeepalives()

	for _ = range time.Tick(5 * time.Second) {
		// do nothing
	}
}
