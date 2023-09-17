package common

import (
	"fmt"
	"io"
	"net"
)

func RecvLong(conn *net.UDPConn) ([]byte, *net.UDPAddr, error) {
	// 创建新文件
	// f, err := os.Create(fileName)
	// if err != nil {
	// 	fmt.Println("Create err:", err)
	// 	return
	// }
	// defer f.Close()
	var ret []byte

	// 接收客户端发送文件内容，原封不动写入文件
	buf := make([]byte, 4096)
	for {
		n, addr, err := conn.ReadFromUDP(buf)
		if err != nil {
			if err == io.EOF {
				fmt.Println("文件接收完毕")
				err = nil
			} else {
				fmt.Println("Read err:", err)
			}
			return ret, addr, err
		}
		ret = append(ret, buf[:n]...)
		// fmt.Println("接收到的数据：", string(buf[:n]), addr)
		// f.Write(buf[:n]) // 写入文件，读多少写多少
	}
}
