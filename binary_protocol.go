package katsubushi

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"reflect"
	"strconv"
	"sync/atomic"
)

const (
	headerSize    = 24
	magicRequest  = 0x80
	magicResponse = 0x81
	opcodeGet     = 0x00
	opcodeVersion = 0x0b
	opcodeStat    = 0x10
)

type bRequest struct {
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

func newBRequest(r io.Reader) (req *bRequest, err error) {
	req = &bRequest{}
	buf := make([]byte, headerSize)
	n, e := io.ReadFull(r, buf)
	if n == 0 {
		return nil, io.EOF
	} else if n < headerSize {
		return nil, fmt.Errorf("binary request header is shorter than %d: %x", headerSize, buf)
	}
	if e != nil {
		return nil, fmt.Errorf("failed to read binary request header: %s", e)
	}

	req.magic = buf[0]
	if req.magic == 0x00 {
		return nil, io.EOF
	} else if req.magic != magicRequest {
		return nil, fmt.Errorf("invalid request magic: %x", req.magic)
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

	if bodyLen < uint32(keyLen)+uint32(extraLen) {
		return nil, fmt.Errorf("total body %d is too small. key length: %d, extra length %d", bodyLen, keyLen, extraLen)
	}

	bodyBuf := make([]byte, bodyLen)
	_, e2 := io.ReadFull(r, bodyBuf)
	if e2 != nil {
		return nil, fmt.Errorf("failed to read binary request body: %s", e2)
	}

	req.extras = bodyBuf[0:extraLen]
	req.key = string(bodyBuf[extraLen : uint16(extraLen)+keyLen])
	req.value = string(bodyBuf[uint16(extraLen)+keyLen : bodyLen])

	return
}

type bResponse struct {
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

type bResponseConfig struct {
	status [2]byte
	cas    [8]byte
	extras []byte
	key    string
	value  string
}

func newBResponse(opcode byte, opaque [4]byte, resConf bResponseConfig) *bResponse {
	var extras []byte
	if resConf.extras == nil {
		extras = []byte{}
	} else {
		extras = resConf.extras
	}

	return &bResponse{
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

func (res bResponse) Bytes() []byte {
	extraLen := len(res.extras)
	keyLen := len(res.key)
	valueLen := len(res.value)
	totalLen := headerSize + extraLen + keyLen + valueLen

	keyLenBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(keyLenBytes, uint16(keyLen))

	extraLenByte := byte(extraLen)

	bodyLenBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(bodyLenBytes, uint32(extraLen+keyLen+valueLen))

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

	copy(data[headerSize:], res.extras)
	copy(data[headerSize+extraLen:], res.key)
	copy(data[headerSize+extraLen+keyLen:], res.value)

	return data
}

// IsBinaryProtocol judges whether a protocol is binary or text
func (app *App) IsBinaryProtocol(r *bufio.Reader) (bool, error) {
	firstByte, err := r.Peek(1)
	if err != nil {
		return false, err
	}
	return firstByte[0] == magicRequest, nil
}

// RespondToBinary responds to a binary request with a binary response.
// A request should be read from r, not conn.
// Because the request reader might be buffered.
func (app *App) RespondToBinary(r io.Reader, conn net.Conn) {
	for {
		app.extendDeadline(conn)

		req, err := newBRequest(r)
		if err != nil {
			if err != io.EOF {
				log.Warn(err)
			}
			return
		}

		cmd, err := app.BytesToBinaryCmd(*req)
		if err != nil {
			if err := app.writeBinaryError(conn); err != nil {
				log.Warnf("error on write error: %s", err)
				return
			}
			continue
		}
		w := bufio.NewWriter(conn)
		if err := cmd.Execute(app, w); err != nil {
			log.Warnf("error on execute cmd %s: %s", cmd, err)
			return
		}
		if err := w.Flush(); err != nil {
			if err != io.EOF {
				log.Warnf("error on cmd %s write: %s", cmd, err)
			}
			return
		}
	}
}

func (app *App) writeBinaryError(w io.Writer) error {
	// TODO: Opcode, Opaque and Status are static and not accurate. It's better to make them dynamic
	// opcode: GET, it should be a requested opcode
	// opaque: zero padding, it should be a requested opaque
	res := newBResponse(opcodeGet, [4]byte{0x00, 0x00, 0x00, 0x00}, bResponseConfig{
		// status: Internal Error, it should be determined by a request or server condition
		status: [2]byte{0x00, 0x84},
	})

	n, err := w.Write(res.Bytes())
	if n < len(respError) {
		return fmt.Errorf("failed to write error response")
	}
	return err
}

// BytesToCmd converts byte array to a MemdBCmd and returns it.
func (app *App) BytesToBinaryCmd(req bRequest) (cmd MemdCmd, err error) {
	switch req.opcode {
	case opcodeGet:
		atomic.AddInt64(&(app.cmdGet), 1)
		cmd = &MemdBCmdGet{
			Name:   "GET",
			Key:    req.key,
			Opaque: req.opaque,
		}
	case opcodeVersion:
		cmd = &MemdBCmdVersion{
			Opaque: req.opaque,
		}
	case opcodeStat:
		cmd = &MemdBCmdStat{
			Key:    req.key,
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
	Key    string
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

	res := newBResponse(opcodeGet, cmd.Opaque, bResponseConfig{
		// fixed 4bytes flags is given to GET response
		extras: []byte{0x00, 0x00, 0x00, 0x00},
		value:  strconv.FormatUint(id, 10),
	})

	_, err2 := w.Write(res.Bytes())
	return err2
}

// MemdBCmdVersion defines binary VERSION command.
type MemdBCmdVersion struct {
	Opaque [4]byte
}

// Execute writes binary Version number.
func (cmd MemdBCmdVersion) Execute(app *App, w io.Writer) error {
	res := newBResponse(opcodeVersion, cmd.Opaque, bResponseConfig{
		value: Version,
	})

	_, err := w.Write(res.Bytes())
	return err
}

// MemdCmdStat defines binary Stat command.
type MemdBCmdStat struct {
	Key    string
	Opaque [4]byte
}

// Execute writes binary stat
// ref. https://github.com/memcached/memcached/wiki/BinaryProtocolRevamped#stat
func (cmd *MemdBCmdStat) Execute(app *App, w io.Writer) error {
	// ignore optional key (items, slabs) for now
	s := app.GetStats()
	statsValue := reflect.ValueOf(s)
	statsType := reflect.TypeOf(s)
	for i := 0; i < statsType.NumField(); i++ {
		field := statsType.Field(i)
		tag := field.Tag.Get("memd")
		if tag == "" {
			continue
		}
		var val string
		v := statsValue.FieldByIndex(field.Index).Interface()
		switch _v := v.(type) {
		case int:
			val = strconv.Itoa(_v)
		case int64:
			val = strconv.FormatInt(int64(_v), 10)
		case string:
			val = string(_v)
		}
		res := newBResponse(opcodeStat, cmd.Opaque, bResponseConfig{
			key:   tag,
			value: val,
		})
		if _, err := w.Write(res.Bytes()); err != nil {
			return err
		}
	}
	// for teminate the sequence
	emptyRes := newBResponse(opcodeStat, cmd.Opaque, bResponseConfig{})
	_, err := w.Write(emptyRes.Bytes())
	return err
}
