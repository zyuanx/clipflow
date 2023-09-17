package main

import (
	"flag"
	"fmt"
	"image"
	"net"
	"os"

	_ "image/jpeg" // 导入JPEG格式支持
	_ "image/png"  // 导入PNG格式支持

	"github.com/zyuanx/clipflow/client"
	"github.com/zyuanx/clipflow/server"
	"golang.design/x/clipboard"
)

func GetClientIPs() ([]*net.IPNet, error) {
	addrs, err := net.InterfaceAddrs()

	if err != nil {
		return nil, err
	}
	var ips []*net.IPNet

	for _, address := range addrs {
		// 检查ip地址判断是否回环地址
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				ips = append(ips, ipnet)
			}
		}
	}

	return ips, nil
}

func readImage() []byte {
	file, err := os.Open("test.jpg")
	if err != nil {
		fmt.Println("无法打开图片文件:", err)
	}
	defer file.Close()

	// 解码图片
	img, _, err := image.Decode(file)
	if err != nil {
		fmt.Println("无法解码图片:", err)
	}

	// 将图片转换为RGBA格式
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	rgba := image.NewRGBA(bounds)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			rgba.Set(x, y, img.At(x, y))
		}
	}

	// 将RGBA格式的图片转换为[]byte
	rgbData := rgba.Pix

	// 打印图片数据长度和内容
	fmt.Println("图片数据长度:", len(rgbData))
	fmt.Println("图片数据内容:", rgbData)
	return rgbData
}

var peer = flag.String("p", "server", "server or client")

func main() {
	// Init returns an error if the package is not ready for use.
	err := clipboard.Init()
	if err != nil {
		panic(err)
	}
	// text := clipboard.Read(clipboard.FmtText)
	// fmt.Println(string(text))

	clipboard.Write(clipboard.FmtImage, readImage())
	flag.Parse()
	fmt.Println("peer:", *peer)
	if *peer == "server" {
		server.Server()
	} else {
		client.Client()
	}
}
