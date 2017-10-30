package network

import (
	"encoding/binary"
	"errors"
	"io"
	"math"
	"bytes"
)

// --------------
// | len | data |
// --------------
// ---------------------
// | data| len | data |
// ---------------------
type MCUMsgParser struct {
	lenMsgLen    int
	lenSeek		int
	minMsgLen    uint32
	maxMsgLen    uint32
	littleEndian bool
}

func NewMCUMsgParser() *MCUMsgParser {
	p := new(MCUMsgParser)
	p.lenMsgLen = 2
	p.lenSeek = 0	//长度字节所在位置位移
	p.minMsgLen = 1
	p.maxMsgLen = 4096
	p.littleEndian = false

	return p
}

// It's dangerous to call the method on reading or writing
func (p *MCUMsgParser) SetMsgLen(lenSeek int,lenMsgLen int, minMsgLen uint32, maxMsgLen uint32) {
	if lenMsgLen == 1 || lenMsgLen == 2 || lenMsgLen == 4 {
		p.lenMsgLen = lenMsgLen
	}
	if lenSeek > 0 {
		p.lenSeek = lenSeek
	}
	if minMsgLen != 0 {
		p.minMsgLen = minMsgLen
	}
	if maxMsgLen != 0 {
		p.maxMsgLen = maxMsgLen
	}

	var max uint32
	switch p.lenMsgLen {
	case 1:
		max = math.MaxUint8
	case 2:
		max = math.MaxUint16
	case 4:
		max = math.MaxUint32
	}
	if p.minMsgLen > max {
		p.minMsgLen = max
	}
	if p.maxMsgLen > max {
		p.maxMsgLen = max
	}
}

// It's dangerous to call the method on reading or writing
func (p *MCUMsgParser) SetByteOrder(littleEndian bool) {
	p.littleEndian = littleEndian
}

/*
这里要加入分包 长度等方法.区别之前的分包方式
p.lenSeek 长度数据的位移
*/
// goroutine safe
func (p *MCUMsgParser) Read(conn *TCPConn) ([]byte, error) {
	var b [p.lenSeek+4]byte
	bufMsgLen := b[:p.lenMsgLen+p.lenSeek]

	// read len
	if _, err := io.ReadFull(conn, bufMsgLen); err != nil {
		return nil, err
	}

	// parse len
	var msgLen uint32
	bufMsgLenT := bufMsgLen[p.lenSeek:] //取出真实的长度
	switch p.lenMsgLen {
	case 1:
		msgLen = uint32(bufMsgLenT[0])
	case 2:
		if p.littleEndian {
			msgLen = uint32(binary.LittleEndian.Uint16(bufMsgLenT))
		} else {
			msgLen = uint32(binary.BigEndian.Uint16(bufMsgLenT))
		}
	case 4:
		if p.littleEndian {
			msgLen = binary.LittleEndian.Uint32(bufMsgLenT)
		} else {
			msgLen = binary.BigEndian.Uint32(bufMsgLenT)
		}
	}

	// check len
	if msgLen > p.maxMsgLen {
		return nil, errors.New("message too long")
	} else if msgLen < p.minMsgLen {
		return nil, errors.New("message too short")
	}

	// data
	msgData := make([]byte, msgLen)
	if _, err := io.ReadFull(conn, msgData); err != nil {
		return nil, err
	}

	return bytes.Join([][]byte{bufMsgLen, msgData}, []byte("")), nil
}

// goroutine safe
//写数据不做处理,直接写(不加入长度数据,长度数据由协议本身提供)
//TODO:此处可以优化
func (p *MCUMsgParser) Write(conn *TCPConn, args ...[]byte) error {
	// get len
	var msgLen uint32
	for i := 0; i < len(args); i++ {
		msgLen += uint32(len(args[i]))
	}

	// check len
	if msgLen > p.maxMsgLen {
		return errors.New("message too long")
	} else if msgLen < p.minMsgLen {
		return errors.New("message too short")
	}

	msg := make([]byte, msgLen)


	// write data
	l := 0
	for i := 0; i < len(args); i++ {
		copy(msg[l:], args[i])
		l += len(args[i])
	}

	conn.Write(msg)

	return nil
}
