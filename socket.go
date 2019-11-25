package socket

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
)

const (
	//LEN message prefix length
	LEN = 4
	//INTERVAL read step length
	INTERVAL = 1024
	//CHANNELSIZE size of channel
	CHANNELSIZE = 16
)

// Stream instance
type Stream struct {
	writeChan chan []byte
	readChan  chan []byte
	//
	streamID int
}

// // Read from stream
// func (s *Stream) Read() []byte {
// 	return <-s.readChan
// }

// Read from stream
func (s *Stream) Read(data []byte) (n int, err error) {
	d := <-s.readChan
	copy(data, d)
	return len(d), nil
}

// Write(b []byte) (n int, err error)
// Write to stream
func (s *Stream) Write(data []byte) (n int, err error) {
	pref := intToBytes(s.streamID)
	var buffer bytes.Buffer
	buffer.Write(pref)
	buffer.Write(data)
	s.writeChan <- buffer.Bytes()
	// WARNING!!! the returned n is not actually bytes number.
	// This is becaulse io.Copy() need `input bytes count` equal `ouput bytes count`.
	return len(data), nil
}

// Socket instance
type Socket struct {
	Closef context.CancelFunc
	Ctx    context.Context
	//
	Conn net.Conn
	//
	writeChan chan []byte
	//
	streamMap map[int]*Stream
}

//
func NewSocket(clf context.CancelFunc, ctx context.Context, conn net.Conn) *Socket {
	socket := Socket{
		Closef:    clf,
		Ctx:       ctx,
		Conn:      conn,
		writeChan: make(chan []byte, CHANNELSIZE),
		streamMap: make(map[int]*Stream),
	}
	return &socket
}

// Start running this socket
func (s *Socket) Start() error {
	if len(s.streamMap) == 0 {
		return errors.New("No Stream has been Registed.")
	}
	go s.writeMsg()
	go s.readMsg()
	return nil
}

// Register stream_id & stream_handler
func (s *Socket) Register(streamid int) (*Stream, error) {
	_, ok := s.streamMap[streamid]
	if ok {
		return nil, errors.New("This stream has been registed.")
	}
	stream := Stream{
		writeChan: s.writeChan,
		readChan:  make(chan []byte, CHANNELSIZE),
		streamID:  streamid,
	}
	s.streamMap[streamid] = &stream
	return &stream, nil
}

// writeMsg wait msg to write.
func (s *Socket) writeMsg() {
	for {
		select {
		case data := <-s.writeChan:
			pref := intToBytes(len(data))
			var buffer bytes.Buffer
			buffer.Write(pref)
			buffer.Write(data)
			_, err := s.Conn.Write(buffer.Bytes())
			if err != nil {
				fmt.Println("Send Error,", err)
			}
		case <-s.Ctx.Done():
			fmt.Println("Quit WriteMsg()")
			return
		}
	}
}

// readMsg wait msg to read.
func (s *Socket) readMsg() {
	tmpBuffer := make([]byte, 0)
	data := make([]byte, INTERVAL)
	for {
		// get length
		n, err := s.Conn.Read(data)
		if err != nil {
			s.Conn.Close()
			fmt.Println("Conn has been Closed.")
			s.Closef()
			break
		}
		tmpBuffer = s.unpack(append(tmpBuffer, data[:n]...))
	}
}

func (s *Socket) unpack(buffer []byte) []byte {
	length := len(buffer)

	var i int
	for i = 0; i < length; i = i + 1 {
		if length < i+LEN {
			break
		}
		messageLength := bytesToInt(buffer[i : i+LEN])
		if length < i+LEN+messageLength {
			break
		}
		data := buffer[i+LEN : i+LEN+messageLength]
		i += LEN + messageLength - 1
		// s.readChan <- data
		streamid := bytesToInt(data[:4])
		stream, ok := s.streamMap[streamid]
		if ok {
			stream.readChan <- data[4:]
		}
	}

	if i == length {
		return make([]byte, 0)
	}
	return buffer[i:]
}

func bytesToInt(b []byte) int {
	bytesBuffer := bytes.NewBuffer(b)

	var x int32
	binary.Read(bytesBuffer, binary.BigEndian, &x)

	return int(x)
}

func intToBytes(n int) []byte {
	x := int32(n)

	bytesBuffer := bytes.NewBuffer([]byte{})
	binary.Write(bytesBuffer, binary.BigEndian, x)
	return bytesBuffer.Bytes()
}
