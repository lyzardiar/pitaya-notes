// Copyright (c) nano Author and TFG Co. All Rights Reserved.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package component

import (
	"context"
	"reflect"
	"unicode"
	"unicode/utf8"

	"github.com/golang/protobuf/proto"
	"github.com/topfreegames/pitaya/conn/message"
)

var (
	typeOfError    = reflect.TypeOf((*error)(nil)).Elem()
	typeOfBytes    = reflect.TypeOf(([]byte)(nil))
	typeOfContext  = reflect.TypeOf(new(context.Context)).Elem()
	typeOfProtoMsg = reflect.TypeOf(new(proto.Message)).Elem()
)

//通过首字母大写判断导出情况
func isExported(name string) bool {
	w, _ := utf8.DecodeRuneInString(name) //返回utf8编码的字符串
	return unicode.IsUpper(w)
}

// isRemoteMethod decide a method is suitable remote method
func isRemoteMethod(method reflect.Method) bool {
	mt := method.Type
	// Method must be exported.
	if method.PkgPath != "" {
		return false
	}

	// Method needs at least two ins: receiver and context.Context
	if mt.NumIn() != 2 && mt.NumIn() != 3 {
		return false
	}

	if t1 := mt.In(1); !t1.Implements(typeOfContext) { //判断第二个参数是否context类型
		return false
	}

	if mt.NumIn() == 3 {
		if t2 := mt.In(2); !t2.Implements(typeOfProtoMsg) { ////判断第三个参数是否proto.Message类型
			return false
		}
	}

	// Method needs two outs: interface{}(that implements proto.Message), error
	if mt.NumOut() != 2 { //判断返回值的数量
		return false
	}

	if (mt.Out(0).Kind() != reflect.Ptr) || mt.Out(1) != typeOfError {
		return false //第一个返回值是指针 第二个返回值是是error
	}

	if o0 := mt.Out(0); !o0.Implements(typeOfProtoMsg) {
		return false //返回值的类型是proto.Message
	}

	return true
}

// isHandlerMethod decide a method is suitable handler method
func isHandlerMethod(method reflect.Method) bool {
	mt := method.Type
	// Method must be exported.
	if method.PkgPath != "" {
		return false
	}

	// Method needs two or three ins: receiver, context.Context and optional []byte or pointer.
	if mt.NumIn() != 2 && mt.NumIn() != 3 { //判断方法的参数 需要有两个或者三个参数 receivcer context 第三个参数是[]byte 或者 *proto.message
		return false
	}

	if t1 := mt.In(1); !t1.Implements(typeOfContext) {
		return false
	}

	if mt.NumIn() == 3 && mt.In(2).Kind() != reflect.Ptr && mt.In(2) != typeOfBytes {
		return false //第三个参数是ptr或者[]byte
	}

	// Method needs either no out or two outs: interface{}(or []byte), error
	if mt.NumOut() != 0 && mt.NumOut() != 2 {
		return false
	}

	if mt.NumOut() == 2 && (mt.Out(1) != typeOfError || mt.Out(0) != typeOfBytes && mt.Out(0).Kind() != reflect.Ptr) {
		return false //返回值有两个 第一个是[]byte或者proto.Message 第二个是error
	}

	return true
}

func suitableRemoteMethods(typ reflect.Type, nameFunc func(string) string) map[string]*Remote {
	methods := make(map[string]*Remote)
	for m := 0; m < typ.NumMethod(); m++ {
		method := typ.Method(m)
		mt := method.Type
		mn := method.Name
		if isRemoteMethod(method) {
			// rewrite remote name
			if nameFunc != nil {
				mn = nameFunc(mn)
			}
			methods[mn] = &Remote{
				Method:  method,
				HasArgs: method.Type.NumIn() == 3,
			}
			if mt.NumIn() == 3 {
				methods[mn].Type = mt.In(2) //方法的第二个输入参数的类型消息的类型参数
			}
		}
	}
	return methods
}

func suitableHandlerMethods(typ reflect.Type, nameFunc func(string) string) map[string]*Handler {
	methods := make(map[string]*Handler)
	//根据方法的数量取出方法
	for m := 0; m < typ.NumMethod(); m++ {
		method := typ.Method(m)
		mt := method.Type //方法的类型信息
		mn := method.Name //方法名
		if isHandlerMethod(method) {
			raw := false
			if mt.NumIn() == 3 && mt.In(2) == typeOfBytes {
				raw = true //是否原始字节消息
			}
			// rewrite handler name
			if nameFunc != nil {
				mn = nameFunc(mn)
			}
			var msgType message.Type
			if mt.NumOut() == 0 {
				msgType = message.Notify //没有返回值的是Nottfy
			} else {
				msgType = message.Request //有返回值的是Request
			}
			handler := &Handler{
				Method:      method,  //方法
				IsRawArg:    raw,     //消息是否未序列化
				MessageType: msgType, //request notify
			}
			if mt.NumIn() == 3 {
				handler.Type = mt.In(2)
			}
			methods[mn] = handler
		}
	}
	return methods
}
