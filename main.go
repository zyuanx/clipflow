package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"

	"golang.design/x/clipboard"
)

var recvMsg []byte
var recvMsgMutex = sync.RWMutex{}

type MessageType int

const (
	MSG_TYPE_TEXT MessageType = iota
	MSG_TYPE_IMAGE
)

type Message struct {
	MsgType MessageType
	Msg     []byte
}

var LocalIP = ""

func GetPublicIP() string {
	conn, _ := net.Dial("udp", "8.8.8.8:80")
	defer conn.Close()
	localAddr := conn.LocalAddr().String()
	idx := strings.LastIndex(localAddr, ":")
	return localAddr[0:idx]
}
func init() {
	err := clipboard.Init()
	if err != nil {
		log.Panicln(err)
	}

	LocalIP = GetPublicIP()
	Discovery()
	log.Printf("Server start in %s\n", LocalIP)

}

func listenClipboard() {
	ctx := context.Background()
	ch1 := clipboard.Watch(ctx, clipboard.FmtText)
	ch2 := clipboard.Watch(ctx, clipboard.FmtImage)
	for {
		select {
		case <-ch1:
			ret := clipboard.Read(clipboard.FmtText)
			if judgeChange(ret) {
				continue
			}
			sendMessage(ret, MSG_TYPE_TEXT)
		case <-ch2:
			ret := clipboard.Read(clipboard.FmtImage)
			if judgeChange(ret) {
				continue
			}
			sendMessage(ret, MSG_TYPE_IMAGE)
		}
	}
}

func judgeChange(ret []byte) bool {
	recvMsgMutex.RLock()
	defer recvMsgMutex.RUnlock()
	return bytes.Equal(recvMsg, ret)
}

func sendMessage(ret []byte, t MessageType) {
	msg := Message{
		MsgType: t,
		Msg:     ret,
	}
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Marshal error: %v\n", err)
		return
	}
	mutex.RLock()
	for _, ser := range LocalServices {
		if ser.IP == LocalIP {
			continue
		}
		req, err := http.NewRequest("POST", fmt.Sprintf("http://%s:%d/receive", ser.IP, ser.Port), bytes.NewReader(data))
		if err != nil {
			log.Printf("NewRequest error: %v\n", err)
			continue
		}
		req.Header.Set("Content-Type", "application/json")
		_, err = http.DefaultClient.Do(req)
		if err != nil {
			log.Printf("Do error: %v\n", err)
			continue
		}
		log.Printf("Send message to %s:%d success\n", ser.IP, ser.Port)
	}
	mutex.RUnlock()
}

func main() {
	go listenClipboard()
	http.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "pong")
	})

	http.HandleFunc("/services", func(w http.ResponseWriter, r *http.Request) {
		mutex.RLock()
		defer mutex.RUnlock()
		for _, v := range LocalServices {
			fmt.Fprintf(w, "%+v\n", v)
		}
	})

	http.HandleFunc("/receive", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var msg Message
		if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}
		log.Printf("Receive message: msg.MsgType %+v\n", msg.MsgType)

		recvMsgMutex.Lock()
		defer recvMsgMutex.Unlock()
		switch msg.MsgType {
		case MSG_TYPE_TEXT:
			clipboard.Write(clipboard.FmtText, msg.Msg)
		case MSG_TYPE_IMAGE:
			clipboard.Write(clipboard.FmtImage, msg.Msg)
		}
		recvMsg = msg.Msg
		fmt.Fprintf(w, "OK")
	})

	http.ListenAndServe(":8080", nil)
}
