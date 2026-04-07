package protocol

import (
	"encoding/binary"
	"fmt"
	"io"
)

type MsgType byte

const (
	MsgAuth     MsgType = 0x01
	MsgAuthOK   MsgType = 0x02
	MsgAuthFail MsgType = 0x03
	MsgInput    MsgType = 0x04
	MsgOutput   MsgType = 0x05
	MsgResize   MsgType = 0x06
)

// WriteMessage writes a framed message: [1 byte type][4 bytes len][N bytes payload]
func WriteMessage(w io.Writer, msgType MsgType, payload []byte) error {
	header := make([]byte, 5)
	header[0] = byte(msgType)
	binary.BigEndian.PutUint32(header[1:], uint32(len(payload)))
	if _, err := w.Write(header); err != nil {
		return err
	}
	if len(payload) > 0 {
		_, err := w.Write(payload)
		return err
	}
	return nil
}

// ReadMessage reads a framed message from the reader.
func ReadMessage(r io.Reader) (MsgType, []byte, error) {
	header := make([]byte, 5)
	if _, err := io.ReadFull(r, header); err != nil {
		return 0, nil, err
	}
	msgType := MsgType(header[0])
	length := binary.BigEndian.Uint32(header[1:])
	if length > 10*1024*1024 {
		return 0, nil, fmt.Errorf("message too large: %d bytes", length)
	}
	payload := make([]byte, length)
	if length > 0 {
		if _, err := io.ReadFull(r, payload); err != nil {
			return 0, nil, err
		}
	}
	return msgType, payload, nil
}

type ResizeMsg struct {
	Rows uint16
	Cols uint16
}

func EncodeResize(rows, cols uint16) []byte {
	b := make([]byte, 4)
	binary.BigEndian.PutUint16(b[0:], rows)
	binary.BigEndian.PutUint16(b[2:], cols)
	return b
}

func DecodeResize(b []byte) ResizeMsg {
	return ResizeMsg{
		Rows: binary.BigEndian.Uint16(b[0:]),
		Cols: binary.BigEndian.Uint16(b[2:]),
	}
}
