package katsubushi

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"strconv"
	"strings"
)

const (
	headerSize    = 24
	magicRequest  = 0x80
	magicResponse = 0x81
	opcodeGet     = 0x00
	opcodeVersion = 0x0b
)

type request struct {
	magic    byte
	opcode   byte
	dataType byte
	vBucket  [2]byte
	opaque   [4]byte
	cas      [8]byte
	extras   []byte
	key      string
	value    string
}

func newRequest(r io.Reader) (req request, err error) {
	req = request{}
	buf := make([]byte, headerSize)
	n, e := io.ReadFull(r, buf)
	if n < headerSize {
		err = fmt.Errorf("binary request header is shorter than %d: %x", headerSize, buf)
	}
	if e != nil {
		err = fmt.Errorf("failed to read binary request header: %s", e)
	}

	req.magic = buf[0]
	if req.magic != magicRequest {
		err = fmt.Errorf("invalid request magic: %x", req.magic)
	}

	req.opcode = buf[1]
	req.dataType = buf[5]
	req.vBucket[0] = buf[6]
	req.vBucket[1] = buf[7]
	req.opaque[0] = buf[12]
	req.opaque[1] = buf[13]
	req.opaque[2] = buf[14]
	req.opaque[3] = buf[15]
	req.cas[0] = buf[16]
	req.cas[1] = buf[17]
	req.cas[2] = buf[18]
	req.cas[3] = buf[19]
	req.cas[4] = buf[20]
	req.cas[5] = buf[21]
	req.cas[6] = buf[22]
	req.cas[7] = buf[23]

	keyLen := binary.BigEndian.Uint16(buf[2:4])
	extraLen := uint8(buf[4])
	bodyLen := binary.BigEndian.Uint32(buf[8:12])

	bodyBuf := make([]byte, bodyLen)
	n2, e2 := io.ReadFull(r, bodyBuf)
	if uint32(n2) < bodyLen {
		err = fmt.Errorf("binary request body is shorter than %d: %x", bodyLen, bodyBuf)
	}
	if e2 != nil {
		err = fmt.Errorf("failed to read binary request body: %s", e2)
	}

	req.extras = bodyBuf[0:extraLen]
	req.key = string(bodyBuf[extraLen : uint16(extraLen)+keyLen])
	req.value = string(bodyBuf[uint16(extraLen)+keyLen : bodyLen])

	return
}

type response struct {
	magic    byte
	opcode   byte
	dataType byte
	status   [2]byte
	opaque   [4]byte
	cas      [8]byte
	extras   []byte
	key      string
	value    string
}

type responseConfig struct {
	status [2]byte
	cas    [8]byte
	extras []byte
	key    string
	value  string
}

func newResponse(opcode byte, opaque [4]byte, resConf responseConfig) response {
	var extras []byte
	if resConf.extras == nil {
		extras = []byte{}
	} else {
		extras = resConf.extras
	}

	return response{
		magic:    magicResponse,
		opcode:   opcode,
		dataType: 0x00, // data type: reserved for future
		status:   resConf.status,
		opaque:   opaque,
		cas:      resConf.cas,
		extras:   extras,
		key:      resConf.key,
		value:    resConf.value,
	}
}

func (res response) toBytes() []byte {
	extraLen := len(res.extras)
	keyLen := len(res.key)
	valueLen := len(res.value)
	totalLen := headerSize + extraLen + keyLen + valueLen

	keyLenBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(keyLenBytes, uint16(keyLen))

	extraLenByte := byte(uint8(extraLen))

	bodyLenBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(bodyLenBytes, uint32(extraLen+valueLen))

	data := make([]byte, totalLen)
	data[0] = res.magic
	data[1] = res.opcode
	data[2] = keyLenBytes[0]
	data[3] = keyLenBytes[1]
	data[4] = extraLenByte
	data[5] = res.dataType
	data[6] = res.status[0]
	data[7] = res.status[1]
	data[8] = bodyLenBytes[0]
	data[9] = bodyLenBytes[1]
	data[10] = bodyLenBytes[2]
	data[11] = bodyLenBytes[3]
	data[12] = res.opaque[0]
	data[13] = res.opaque[1]
	data[14] = res.opaque[2]
	data[15] = res.opaque[3]
	data[16] = res.cas[0]
	data[17] = res.cas[1]
	data[18] = res.cas[2]
	data[19] = res.cas[3]
	data[20] = res.cas[4]
	data[21] = res.cas[5]
	data[22] = res.cas[6]
	data[23] = res.cas[7]

	for i := 0; i < extraLen; i++ {
		data[i+headerSize] = res.extras[i]
	}

	for i := 0; i < keyLen; i++ {
		data[i+headerSize+extraLen] = res.key[i]
	}

	for i := 0; i < valueLen; i++ {
		data[i+headerSize+extraLen+keyLen] = res.value[i]
	}

	return data
}

// IsBinaryProtocol judges whether a protocol is binary or text
func (app *App) IsBinaryProtocol(r *bufio.Reader) (bool, error) {
	firstByte, err := r.Peek(1)
	return firstByte[0] == magicRequest, err
}

// RespondToBinary responds to a binary request with a binary response.
func (app *App) RespondToBinary(r io.Reader, w io.Writer) {
	for {
		req, err := newRequest(r)
		if err != nil {
			log.Warn(err)
			return
		}

		cmd, err := app.BytesToBinaryCmd(req)
		if err != nil {
			if err := app.writeBinaryError(w); err != nil {
				log.Warn("error on write error: %s", err)
				return
			}
			continue
		}
		w := bufio.NewWriter(w)
		if err := cmd.Execute(app, w); err != nil {
			log.Warn("error on execute cmd %s: %s", cmd, err)
			return
		}
		if err := w.Flush(); err != nil {
			if err != io.EOF {
				log.Warn("error on cmd %s write: %s", cmd, err)
			}
			return
		}
	}
}

func (app *App) writeBinaryError(w io.Writer) error {
	// TODO: Opcode, Opaque and Status are static and not accurate. It's better to make them dynamic
	// opcode: GET, it should be a requested opcode
	// opaque: zero padding, it should be a requested opaque
	res := newResponse(opcodeGet, [4]byte{0x00, 0x00, 0x00, 0x00}, responseConfig{
		// status: Internal Error, it should be determined by a request or server condition
		status: [2]byte{0x00, 0x84},
	})

	n, err := w.Write(res.toBytes())
	if n < len(respError) {
		return fmt.Errorf("failed to write error response")
	}
	return err
}

// BytesToCmd converts byte array to a MemdBCmd and returns it.
func (app *App) BytesToBinaryCmd(req request) (cmd MemdCmd, err error) {
	switch req.opcode {
	case opcodeGet:
		cmd = &MemdBCmdGet{
			Name:   "GET",
			Keys:   strings.Fields(req.key),
			Opaque: req.opaque,
		}
	case opcodeVersion:
		cmd = &MemdBCmdVersion{
			Opaque: req.opaque,
		}
	default:
		err = fmt.Errorf("unknown binary command: %x", req.opcode)
	}
	return
}

// MemdCmdGet defines binary Get command.
type MemdBCmdGet struct {
	Name   string
	Keys   []string
	Opaque [4]byte
}

// Execute generates new ID.
func (cmd *MemdBCmdGet) Execute(app *App, w io.Writer) error {
	id, err := app.NextID()
	if err != nil {
		log.Warn(err)
		if err = app.writeError(w); err != nil {
			log.Warn("error on write error: %s", err)
			return err
		}
		return nil
	}
	log.Debugf("Generated ID: %d", id)

	res := newResponse(opcodeGet, cmd.Opaque, responseConfig{
		// fixed 4bytes flags is given to GET response
		extras: []byte{0x00, 0x00, 0x00, 0x00},
		value:  strconv.FormatUint(id, 10),
	})

	_, err2 := w.Write(res.toBytes())
	return err2
}

// MemdBCmdVersion defines binary VERSION command.
type MemdBCmdVersion struct {
	Opaque [4]byte
}

// Execute writes binary Version number.
func (cmd MemdBCmdVersion) Execute(app *App, w io.Writer) error {
	res := newResponse(opcodeVersion, cmd.Opaque, responseConfig{
		value: Version,
	})

	_, err := w.Write(res.toBytes())
	return err
}
