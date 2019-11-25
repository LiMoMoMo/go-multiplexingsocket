package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"

	socket "github.com/LiMoMoMo/go-multiplexingsocket"
)

const (
	addr = "0.0.0.0:4242"
)

// FileInfo transfer file's info
type FileInfo struct {
	Name string
	Size int64
}

func main() {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		fmt.Println(err)
		return
	}
	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println(err)
			return
		}
		ctx, cancel := context.WithCancel(context.Background())
		socket := socket.NewSocket(cancel, ctx, conn)
		stream1, _ := socket.Register(1)
		stream2, _ := socket.Register(2)
		stream3, _ := socket.Register(3)
		socket.Start()
		// receive file data
		go receiveFile(stream1)
		go receiveFile(stream2)
		go receiveFile(stream3)
		// go receiveData(stream)
	}
}

func receiveFile(stream io.ReadWriter) {
	// get file name
	data := make([]byte, 512)
	n, err := stream.Read(data)
	if err != nil {
		fmt.Println(err)
		return
	}
	info := FileInfo{}
	json.Unmarshal(data[:n], &info)
	fmt.Printf("Got %d bytes, value is: %s\n", n, info.Name)

	// receive file data
	fi, err := os.Create(info.Name)
	if err != nil {
		fmt.Println("create file error:", err)
		return
	}
	defer fi.Close()
	// io.Copy(fi, stream)
	rdata := make([]byte, 32*1024)
	var count int64
	count = 0
	for count < info.Size {
		n, err := stream.Read(rdata)
		if err != nil {
			if err != io.EOF {
				fmt.Printf("Read data error: %v\n", err)
			}
			break
		}
		count += int64(n)
		if _, err = fi.Write(rdata[:n]); err != nil {
			fmt.Printf("Write file error: %v\n", err)
			break
		}
	}

	fmt.Println("Transfer finished.")
	// finish
	_, err = stream.Write([]byte("This is quicServer, transfer finish."))
	if err != nil {
		fmt.Println(err)
		return
	}
}
