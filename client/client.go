// Netcat is a simple read/write client for TCP servers.
package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"time"
)

func main() {
	conn, err := net.Dial("tcp", "localhost:8000")
	if err != nil {
		log.Fatal(err)
	}
	done := make(chan struct{})
	go func() {
		for {
			data := make([]byte, 1024)
			conn.Read(data)
			fmt.Println(string(data))
		}
		io.Copy(os.Stdout, conn) // NOTE: ignoring errors
		log.Println("done")
		done <- struct{}{} // signal the main goroutine
	}()
	// mustCopy(conn, os.Stdin)
	go func() {
		for {
			data := []byte("hello")
			// fmt.Scanln(&data)
			conn.Write(data)
			time.Sleep(1 * time.Second)
		}
	}()
	defer conn.Close()
	<-done // wait for background goroutine to finish
}

func mustCopy(dst io.Writer, src io.Reader) {
	if _, err := io.Copy(dst, src); err != nil {
		log.Fatal(err)
	}
}
