package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"path"
	"time"

	socket "github.com/LiMoMoMo/go-multiplexingsocket"
	pb "github.com/cheggaaa/pb/v3"
)

const (
	addr = "192.168.31.123:4242"
	// addr = "192.168.31.48:4242"
)

// FileInfo transfer file's info
type FileInfo struct {
	Name string
	Size int64
}

type Speed struct {
	bar *pb.ProgressBar
}

func NewSpeed(bar *pb.ProgressBar) *Speed {
	s := &Speed{
		bar: bar,
	}
	return s
}

func (s *Speed) Write(p []byte) (int, error) {
	size := int64(len(p))
	if s.bar != nil {
		s.bar.Add64(size)
	}
	return len(p), nil
}

func (s *Speed) Close() error {
	s.bar.Finish()
	return nil
}

func main() {
	conn, err := net.DialTimeout("tcp", addr, time.Second*5)
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
	// send file
	go sendFile("D:/temp/Tom.mp4", stream1)
	go sendFile("D:/temp/S03E01.mp4", stream2)
	go sendFile("D:/temp/111.mp4", stream3)
	//
	select {}
}

func sendFile(filepath string, stream io.Writer) {
	basePath := path.Base(filepath)
	stat, _ := os.Stat(filepath)

	bar := pb.Start64(stat.Size())
	speed := NewSpeed(bar)
	// defer
	defer func() {
		bar.Finish()
	}()
	info := FileInfo{
		Name: basePath,
		Size: stat.Size(),
	}

	array, err := json.Marshal(info)
	if err != nil {
		fmt.Println("Marshal Error", err)
		return
	}

	_, err = stream.Write(array)
	if err != nil {
		fmt.Printf("write filename error: %v\n", err)
		return
	}

	fp, err := os.Open(filepath)
	if err != nil {
		fmt.Printf("open file %q error: %v\n", filepath, err)
		return
	}
	defer fp.Close()
	_, err = io.Copy(io.MultiWriter(stream, speed), fp)
	if err != nil {
		if err != io.EOF {
			fmt.Println(err)
			return
		}
	}
}
