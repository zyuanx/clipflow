package main

import (
	"encoding/json"
	"log"
	"net"
	"sync"
	"time"

	"golang.org/x/net/ipv4"
)

const (
	multicastAddr = "224.0.1.251"
	multicastPort = 5352
)

var LocalServices map[string]*HeartbeatMessage = make(map[string]*HeartbeatMessage)

var mutex = sync.RWMutex{}

type HeartbeatMessage struct {
	IP   string
	Port int
}

const HeartbeatPort = 8080

func SendHeartbeat(conn *net.UDPConn, addr net.Addr) {
	msg := HeartbeatMessage{
		IP:   LocalIP,
		Port: HeartbeatPort,
	}
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Marshal error: %v\n", err)
		return
	}
	// send heartbeat every 1 seconds
	ticker := time.NewTicker(1 * time.Second)
	for range ticker.C {
		_, err := conn.WriteToUDP(data, addr.(*net.UDPAddr))
		if err != nil {
			log.Printf("Write error: %v\n", err)
		}
	}
}

func ReceiveHeartbeat(conn *net.UDPConn) {
	buf := make([]byte, 1024)
	ticker := time.NewTicker(1 * time.Second)
	for range ticker.C {
		n, addr, err := conn.ReadFromUDP(buf)
		if err != nil {
			log.Printf("Read error: %v\n", err)
			return
		}
		msg := HeartbeatMessage{}
		err = json.Unmarshal(buf[:n], &msg)
		if err != nil {
			log.Printf("Unmarshal error: %v\n", err)
			return
		}
		mutex.Lock()
		LocalServices[addr.String()] = &msg
		mutex.Unlock()
		// log.Printf("Receive heartbeat from %v\n", addr)
	}
}

func Discovery() {
	ipv4Addr := &net.UDPAddr{IP: net.ParseIP(multicastAddr), Port: multicastPort}
	conn, err := net.ListenUDP("udp4", ipv4Addr)
	if err != nil {
		log.Fatalf("ListenUDP error %v\n", err)
	}

	pc := ipv4.NewPacketConn(conn)
	inter, err := net.Interfaces()
	if err != nil {
		log.Fatalf("Get interface error %v\n", err)
	}
	for _, v := range inter {
		ifi, err := net.InterfaceByName(v.Name)
		if err != nil {
			log.Printf("can't find specified interface %v\n", err)
			continue
		}
		if err := pc.JoinGroup(ifi, &net.UDPAddr{IP: net.ParseIP(multicastAddr)}); err != nil {
			log.Printf("JointGroup error %v\n", err)
			continue
		}

		if loop, err := pc.MulticastLoopback(); err == nil {
			if !loop {
				if err := pc.SetMulticastLoopback(true); err != nil {
					log.Printf("SetMulticastLoopback error: %v\n", err)
				}
			}
		}

		go SendHeartbeat(conn, ipv4Addr)

		go ReceiveHeartbeat(conn)
	}
}
