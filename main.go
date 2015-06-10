// receiver project main.go
package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
)

func checkerr(err error) {
	if err != nil {
		log.Fatal("[Red] ", err)
	}
}

type ShareFileInfo struct {
	Path string
	Name string
	Size int64
}

func main() {
	path := "."
	addr := ""

	// 已接收的文件大小
	receivedLength := 0

	// 是否替换"\"为"/"，用于windows系统共享文件到类unix系统
	replaceSlash := false

	if len(os.Args) < 2 {
		fmt.Println("[Red] 参数错误，请输入sender地址")
		return
	}

	addr = os.Args[1]

	if len(os.Args) > 2 {
		path = os.Args[2]
	}

	tcpAddr, err := net.ResolveTCPAddr("tcp4", addr)
	checkerr(err)
	conn, err := net.DialTCP("tcp", nil, tcpAddr)
	checkerr(err)

	// 替换slash需要额外加一个true参数
	if len(os.Args) > 3 && os.Args[3] == "true" {
		replaceSlash = true
	}

	for {
		// 接收一个int值，表示json文件的字节长度
		intlen := 4
		intsum := 0
		buffer := make([]byte, intlen)
		intdata := make([]byte, intlen)
		for intsum < intlen {
			// 剩余长度小于缓冲区长度时，重新创建缓冲区，防止粘包
			sub := intlen - intsum
			if sub < intlen {
				buffer = make([]byte, sub)
			}
			n, err := conn.Read(buffer)
			if err != nil {
				fmt.Println("[Red] socket closed: ", err)
				if err.Error() == "EOF" {
					fmt.Println("[Red] Maybe Succeed.")
					fmt.Println("[Red] Received Size:", receivedLength, "byte")
				}
				return
			}
			intdata = append(intdata[:intsum], buffer[:n]...)
			intsum += n
		}

		// 接收json
		jsonlen := binary.BigEndian.Uint32(intdata[:intlen])
		jsonsum := uint32(0)
		buffer = make([]byte, jsonlen)
		jsondata := make([]byte, jsonlen)
		for jsonsum < jsonlen {
			// 剩余长度小于缓冲区长度时，重新创建缓冲区，防止粘包
			sub := jsonlen - jsonsum
			if sub < jsonlen {
				buffer = make([]byte, sub)
			}
			n, err := conn.Read(buffer)
			if err != nil {
				fmt.Println("[Red] socket closed: ", err)
				return
			}
			jsondata = append(jsondata[:jsonsum], buffer[:n]...)
			jsonsum += uint32(n)
		}

		//解析json
		var shareFileInfo ShareFileInfo
		err = json.Unmarshal(jsondata[:jsonlen], &shareFileInfo)
		checkerr(err)

		// 是否替换slash
		if replaceSlash {
			shareFileInfo.Path = strings.Replace(shareFileInfo.Path, "\\", "/", -1)
		}

		//根据json描述，创建目录，写入文件
		dirInfo, err := os.Stat(shareFileInfo.Path)
		if err != nil || !dirInfo.IsDir() {
			os.MkdirAll(path+shareFileInfo.Path, 0700)
		}
		nativeFilePath := path + shareFileInfo.Path + "/" + shareFileInfo.Name
		fmt.Println("[Red] handle: ", nativeFilePath, shareFileInfo.Size/1024, "KB")
		file, err := os.Create(nativeFilePath)
		bufferlen := 1024 * 1024
		sum := int64(0)
		buffer = make([]byte, bufferlen)
		for sum < shareFileInfo.Size {
			// 剩余长度小于缓冲区长度时，重新创建缓冲区，防止粘包
			sub := shareFileInfo.Size - sum
			if sub < int64(bufferlen) {
				buffer = make([]byte, sub)
			}
			n, err := conn.Read(buffer)
			if err != nil {
				fmt.Println("[Red] socket closed: ", err)
				return
			}
			file.Write(buffer[:n])
			receivedLength += n
			sum += int64(n)
		}
		fmt.Println("[Red] received:", receivedLength/1024/1024, "MB")
	}
}
