package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"net"

	"golang.design/x/clipboard"
	"golang.org/x/net/ipv4"
)

const (
	multicastAddr = "224.0.1.251"
	multicastPort = 5352
)

var recvMsg []byte

type MessageType int

const (
	MSG_TYPE_TEXT MessageType = iota + 1
	MSG_TYPE_IMAGE
)

type Message struct {
	MsgType MessageType
	Msg     []byte
}

const (
	MSG_LENGTH = 8
	MSG_BUF    = 1024
)

var LocalIP = ""

func init() {
	err := clipboard.Init()
	if err != nil {
		log.Panicln(err)
	}
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		fmt.Println("Error getting interface addresses:", err)
		return
	}

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				LocalIP = ipnet.IP.String()
			}
		}
	}
	log.Printf("Server start in %s\n", LocalIP)
}

func listenClipboard(conn *net.UDPConn, addr net.Addr) {
	ch1 := clipboard.Watch(context.TODO(), clipboard.FmtText)
	ch2 := clipboard.Watch(context.TODO(), clipboard.FmtImage)
	go func() {
		for {
			select {
			case <-ch1:
				ret := clipboard.Read(clipboard.FmtText)
				if bytes.Equal(ret, recvMsg) {
					continue
				}
				msg := Message{
					MsgType: MSG_TYPE_TEXT,
					Msg:     ret,
				}
				sendMessage(conn, addr, msg)
			case <-ch2:
				ret := clipboard.Read(clipboard.FmtImage)
				if bytes.Equal(ret, recvMsg) {
					continue
				}
				msg := Message{
					MsgType: MSG_TYPE_IMAGE,
					Msg:     ret,
				}
				sendMessage(conn, addr, msg)
			}
		}
	}()
}

func sendMessage(conn *net.UDPConn, addr net.Addr, msg Message) {

	messageBytes, _ := json.Marshal(msg)
	messageLength := len(messageBytes)
	lengthBytes := make([]byte, MSG_LENGTH)
	binary.BigEndian.PutUint32(lengthBytes, uint32(messageLength))
	messageWithLength := append(lengthBytes, messageBytes...)
	for len(messageWithLength) > 0 {
		mi := min(MSG_BUF, len(messageWithLength))
		buf := messageWithLength[:mi]
		_, _, err := conn.WriteMsgUDP(buf, nil, addr.(*net.UDPAddr))
		if err != nil {
			log.Printf("Write failed, %v\n", err)
		}
		messageWithLength = messageWithLength[mi:]
	}
}

func receiveMessage(conn *net.UDPConn, name string) {
	receivedData := []byte{}
	for {
		buf := make([]byte, MSG_BUF)
		n, addr, err := conn.ReadFromUDP(buf)
		if err != nil {
			log.Printf("Read error, %v\n", err)
		}
		if addr.IP.String() == LocalIP {
			continue
		}
		receivedData = append(receivedData, buf[:n]...)

		for {
			if len(receivedData) < MSG_LENGTH {
				break
			}
			messageLength := int(binary.BigEndian.Uint32(receivedData[:MSG_LENGTH]))
			if len(receivedData) < MSG_LENGTH+messageLength {
				break
			}

			messageBytes := receivedData[MSG_LENGTH : MSG_LENGTH+messageLength]

			var msg Message
			if err := json.Unmarshal(messageBytes, &msg); err != nil {
				fmt.Println("Error deserializing message:", err)
			} else {
				recvMsg = msg.Msg
				switch msg.MsgType {
				case MSG_TYPE_TEXT:
					clipboard.Write(clipboard.FmtText, recvMsg)
				case MSG_TYPE_IMAGE:
					clipboard.Write(clipboard.FmtImage, recvMsg)
				}
				log.Printf("Received from multicast group(%s, %s)\n", addr, name)
			}
			receivedData = receivedData[MSG_LENGTH+messageLength:]
		}

	}
}

func main() {
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

		// go sendMessage(conn, ipv4Addr)
		go listenClipboard(conn, ipv4Addr)

		go receiveMessage(conn, v.Name)
	}
	select {}

}
