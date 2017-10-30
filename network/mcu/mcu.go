package mcu

import (
	"errors"
	"github.com/ihxzihxz/leaf/chanrpc"
	"github.com/ihxzihxz/leaf/log"
	"reflect"
)

/*
本插件由ihxzihxz 38612744@qq.com开发,用于对mcu单片机上通过wifi 4g NBIot上传的数据来进行处理
本插件主要为16进制协议解析并绑定路由提供支持
主要差异参看Route
*/

type Processor struct {
	msgInfo map[string]*MsgInfo
}

type MsgInfo struct {
	msgType       reflect.Type //struct类型
	msgRouter     *chanrpc.Server
	msgHandler    MsgHandler
	msgRawHandler MsgHandler
	msgProtocol   [][]byte
}

type MsgHandler func([]interface{})

type MsgRaw struct {
	msgID      string
	msgRawData []byte
}

func NewProcessor() *Processor {
	p := new(Processor)
	p.msgInfo = make(map[string]*MsgInfo)
	return p
}

// It's dangerous to call the method on routing or marshaling (unmarshaling)
func (p *Processor) Register(msg interface{}, protocol [][]byte) string {
	msgType := reflect.TypeOf(msg)
	if msgType == nil || msgType.Kind() != reflect.Ptr {
		log.Fatal("Register:mcu message pointer required")
	}
	msgID := msgType.Elem().Name()
	if msgID == "" {
		log.Fatal("unnamed mcu message")
	}
	if _, ok := p.msgInfo[msgID]; ok {
		log.Fatal("Register:message %v is already registered", msgID)
	}

	i := new(MsgInfo)
	i.msgType = msgType
	i.msgProtocol = protocol //带入协议供router检验用
	p.msgInfo[msgID] = i
	return msgID
}

// It's dangerous to call the method on routing or marshaling (unmarshaling)
func (p *Processor) SetRouter(msg interface{}, msgRouter *chanrpc.Server) {
	msgType := reflect.TypeOf(msg)
	if msgType == nil || msgType.Kind() != reflect.Ptr {
		log.Fatal("SetRouter:mcu message pointer required")
	}
	msgID := msgType.Elem().Name()
	i, ok := p.msgInfo[msgID]
	if !ok {
		log.Fatal("SetRouter:mcu %v not registered", msgID)
	}

	i.msgRouter = msgRouter
}

// goroutine safe
func (p *Processor) Route(msg interface{}, userData interface{}) error {
	//msg 结构体内容
	//msg不再是结构体而是[]byte类型
	//userData goroutine agent
	value, ok := msg.([]byte)
	if !ok {
		return errors.New("Route:mcu input data type error or empty!")
	}

	for _, data := range p.msgInfo {//所有路由
		for k := 0; k < len(data.msgProtocol); k++ {//每个路由可以有多条mask
			i := 0
			dmin:=0
			if len(value) > len(data.msgProtocol[k]){
				dmin = len(data.msgProtocol[k])
			}else{
				dmin =len(value)
			}

			for ; i < dmin; i++ {//每个字节对比
				if data.msgProtocol[k][i] == 0 {
					continue
				}
				if value[i] != data.msgProtocol[k][i] {
					continue
				}
			}
			//协议mask符合
			if i >= len(value) {
				if data.msgRouter != nil {
					msgbuild := reflect.New(data.msgType.Elem())
					sliceValue := reflect.ValueOf(msg)
					msgbuild.FieldByName("data").Set(sliceValue)
					data.msgRouter.Go(data.msgType, msgbuild, userData)//异步调用
					log.Debug("Route:%v==%v", data.msgType.Elem().Name(),msg)
				}
			}
		}
	}
	return nil
}

/*
中间原始数据不作转换
*/
// goroutine safe
func (p *Processor) Unmarshal(data []byte) (interface{}, error) {
	return data, nil
}

/*
中间原始数据不作转换
*/
// goroutine safe
func (p *Processor) Marshal(msg interface{}) ([][]byte, error) {

	if value, ok := msg.([]byte); ok {
		return [][]byte{value}, nil
	} else {
		return nil, errors.New("Marshal:mcu input data type error or empty!")
	}

}
