package websocket

import (
	"bufio"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
)

// Conn represents a minimal WebSocket connection capable of handling
// text frames. It implements a subset of RFC6455 sufficient for this
// prototype server.
type Conn struct {
	conn net.Conn
	rw   *bufio.ReadWriter
}

// Upgrade performs the HTTP upgrade handshake and returns a Conn.
func Upgrade(w http.ResponseWriter, r *http.Request) (*Conn, error) {
	if !strings.Contains(strings.ToLower(r.Header.Get("Connection")), "upgrade") {
		return nil, fmt.Errorf("websocket: missing upgrade request")
	}
	if strings.ToLower(r.Header.Get("Upgrade")) != "websocket" {
		return nil, fmt.Errorf("websocket: unsupported upgrade protocol")
	}

	key := r.Header.Get("Sec-WebSocket-Key")
	if key == "" {
		return nil, fmt.Errorf("websocket: missing key")
	}

	const acceptGUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"
	sum := sha1.Sum([]byte(key + acceptGUID))
	accept := base64.StdEncoding.EncodeToString(sum[:])

	hijacker, ok := w.(http.Hijacker)
	if !ok {
		return nil, fmt.Errorf("websocket: response writer cannot hijack connection")
	}

	conn, rw, err := hijacker.Hijack()
	if err != nil {
		return nil, fmt.Errorf("websocket: hijack failed: %w", err)
	}

	if _, err := rw.WriteString("HTTP/1.1 101 Switching Protocols\r\n"); err != nil {
		conn.Close()
		return nil, err
	}
	if _, err := rw.WriteString("Upgrade: websocket\r\n"); err != nil {
		conn.Close()
		return nil, err
	}
	if _, err := rw.WriteString("Connection: Upgrade\r\n"); err != nil {
		conn.Close()
		return nil, err
	}
	if _, err := rw.WriteString("Sec-WebSocket-Accept: " + accept + "\r\n\r\n"); err != nil {
		conn.Close()
		return nil, err
	}
	if err := rw.Flush(); err != nil {
		conn.Close()
		return nil, err
	}

	return &Conn{conn: conn, rw: rw}, nil
}

// Close closes the underlying network connection.
func (c *Conn) Close() error {
	return c.conn.Close()
}

// ReadMessage returns the next opcode and payload from the client.
func (c *Conn) ReadMessage() (int, []byte, error) {
	header := make([]byte, 2)
	if _, err := io.ReadFull(c.rw, header); err != nil {
		return 0, nil, err
	}

	fin := header[0]&0x80 != 0
	opcode := int(header[0] & 0x0F)
	if !fin {
		return 0, nil, fmt.Errorf("websocket: fragmented frames are not supported")
	}

	mask := header[1]&0x80 != 0
	payloadLen := int(header[1] & 0x7F)

	switch payloadLen {
	case 126:
		ext := make([]byte, 2)
		if _, err := io.ReadFull(c.rw, ext); err != nil {
			return 0, nil, err
		}
		payloadLen = int(ext[0])<<8 | int(ext[1])
	case 127:
		ext := make([]byte, 8)
		if _, err := io.ReadFull(c.rw, ext); err != nil {
			return 0, nil, err
		}
		payloadLen = 0
		for _, b := range ext {
			payloadLen = (payloadLen << 8) | int(b)
		}
	}

	var maskKey [4]byte
	if mask {
		if _, err := io.ReadFull(c.rw, maskKey[:]); err != nil {
			return 0, nil, err
		}
	}

	payload := make([]byte, payloadLen)
	if _, err := io.ReadFull(c.rw, payload); err != nil {
		return 0, nil, err
	}

	if mask {
		for i := 0; i < payloadLen; i++ {
			payload[i] ^= maskKey[i%4]
		}
	}

	return opcode, payload, nil
}

// WriteMessage writes a single frame with the provided opcode and payload.
func (c *Conn) WriteMessage(opcode int, payload []byte) error {
	header := []byte{0x80 | byte(opcode)}
	length := len(payload)

	switch {
	case length <= 125:
		header = append(header, byte(length))
	case length <= 65535:
		header = append(header, 126, byte(length>>8), byte(length))
	default:
		header = append(header, 127)
		for i := 7; i >= 0; i-- {
			header = append(header, byte(length>>(uint(i)*8)))
		}
	}

	if _, err := c.rw.Write(header); err != nil {
		return err
	}
	if _, err := c.rw.Write(payload); err != nil {
		return err
	}
	return c.rw.Flush()
}
