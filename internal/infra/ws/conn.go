package ws

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/poyaz/reverse-ws-modifier/internal/domain"
	"io"
	"math"
	"net/http"
	"unicode/utf8"
)

const bufferSize = 4096

var closeCodes map[int]string = map[int]string{
	1000: "NormalError",
	1001: "GoingAwayError",
	1002: "ProtocolError",
	1003: "UnknownType",
	1007: "TypeError",
	1008: "PolicyError",
	1009: "MessageTooLargeError",
	1010: "ExtensionError",
	1011: "UnexpectedError",
}

type closeConn interface {
	Close() error
}

type wsConn struct {
	conn   closeConn
	bufrw  *bufio.ReadWriter
	header http.Header
	status uint16
}

func (ws *wsConn) read(size int) ([]byte, error) {
	data := make([]byte, 0)
	for {
		if len(data) == size {
			break
		}
		// Temporary slice to read chunk
		sz := bufferSize
		remaining := size - len(data)
		if sz > remaining {
			sz = remaining
		}
		temp := make([]byte, sz)

		n, err := ws.bufrw.Read(temp)
		if err != nil && err != io.EOF {
			return data, err
		}

		data = append(data, temp[:n]...)
	}

	return data, nil
}

func (ws *wsConn) write(data []byte) error {
	if _, err := ws.bufrw.Write(data); err != nil {
		return err
	}
	return ws.bufrw.Flush()
}

func (ws *wsConn) validate(frame *domain.Frame) error {
	if !frame.IsMasked {
		ws.status = 1002
		return errors.New("protocol error: unmasked client Frame")
	}
	if frame.IsControl() && (frame.Length > 125 || frame.IsFragment) {
		ws.status = 1002
		return errors.New("protocol error: all control frames MUST have a payload length of 125 bytes or less and MUST NOT be fragmented")
	}
	if frame.HasReservedOpcode() {
		ws.status = 1002
		return errors.New("protocol error: opcode " + fmt.Sprintf("%x", frame.Opcode) + " is reserved")
	}
	if frame.Reserved > 0 {
		ws.status = 1002
		return errors.New("protocol error: RSV " + fmt.Sprintf("%x", frame.Reserved) + " is reserved")
	}
	if frame.Opcode == 1 && !frame.IsFragment && !utf8.Valid(frame.Payload) {
		ws.status = 1007
		return errors.New("wrong code: invalid UTF-8 text message ")
	}
	if frame.Opcode == 8 {
		if frame.Length >= 2 {
			code := binary.BigEndian.Uint16(frame.Payload[:2])
			reason := utf8.Valid(frame.Payload[2:])
			if code >= 5000 || (code < 3000 && closeCodes[int(code)] == "") {
				ws.status = 1002
				return errors.New(closeCodes[1002] + " Wrong Code")
			}
			if frame.Length > 2 && !reason {
				ws.status = 1007
				return errors.New(closeCodes[1007] + " invalid UTF-8 reason message")
			}
		} else if frame.Length != 0 {
			ws.status = 1002
			return errors.New(closeCodes[1002] + " Wrong Code")
		}
	}
	return nil
}

// recv receives data and returns a Frame
func (ws *wsConn) recv() (domain.Frame, error) {
	f := domain.Frame{}
	head, err := ws.read(2)
	if err != nil {
		return f, err
	}

	f.IsFragment = (head[0] & 0x80) == 0x00
	f.Opcode = domain.OpcodeType(head[0] & 0x0F)
	f.Reserved = (head[0] & 0x70)

	f.IsMasked = (head[1] & 0x80) == 0x80

	var length uint64
	length = uint64(head[1] & 0x7F)

	if length == 126 {
		data, err := ws.read(2)
		if err != nil {
			return f, err
		}
		length = uint64(binary.BigEndian.Uint16(data))
	} else if length == 127 {
		data, err := ws.read(8)
		if err != nil {
			return f, err
		}
		length = uint64(binary.BigEndian.Uint64(data))
	}
	mask, err := ws.read(4)
	if err != nil {
		return f, err
	}
	f.Length = length

	payload, err := ws.read(int(length)) // possible data loss
	if err != nil {
		return f, err
	}

	for i := uint64(0); i < length; i++ {
		payload[i] ^= mask[i%4]
	}
	f.Payload = payload
	err = ws.validate(&f)
	return f, err
}

// send sends a Frame
func (ws *wsConn) send(frame domain.Frame) error {
	data := make([]byte, 2)
	data[0] = 0x80 | byte(frame.Opcode)
	if frame.IsFragment {
		data[0] &= 0x7F
	}

	if frame.Length <= 125 {
		data[1] = byte(frame.Length)
		data = append(data, frame.Payload...)
	} else if frame.Length > 125 && float64(frame.Length) < math.Pow(2, 16) {
		data[1] = byte(126)
		size := make([]byte, 2)
		binary.BigEndian.PutUint16(size, uint16(frame.Length))
		data = append(data, size...)
		data = append(data, frame.Payload...)
	} else if float64(frame.Length) >= math.Pow(2, 16) {
		data[1] = byte(127)
		size := make([]byte, 8)
		binary.BigEndian.PutUint64(size, frame.Length)
		data = append(data, size...)
		data = append(data, frame.Payload...)
	}
	return ws.write(data)
}

// close sends close Frame and closes the TCP connection
func (ws *wsConn) close() error {
	f := domain.Frame{}
	f.Opcode = 8
	f.Length = 2
	f.Payload = make([]byte, 2)
	binary.BigEndian.PutUint16(f.Payload, ws.status)
	if err := ws.send(f); err != nil {
		return err
	}
	return ws.conn.Close()
}
