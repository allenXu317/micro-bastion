package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
)

func handleRequest(w http.ResponseWriter, r *http.Request) {
	//有个现像，当url为：localhost:8888时，会出现多个报错信息，说明go的listen是一个连接对应多个请求
    //所以我给提了个issue并且提出了修改得意见
	if r.URL.Path == "/" {
		fmt.Fprintln(w, "Bastion is up ")
		return
	}
	r.URL = calculateURL(r)
	//若新URL的HOST为空则直接返回结束进程
	if r.URL.Host == "" {
		log.Println("Requesting nothing")
		return
	}
	r.Host = r.URL.Host
	log.Println("Requesting ", r.URL)
    //发起请求
    //开始转发
	resp, err := http.DefaultTransport.RoundTrip(r)

	if err != nil {
		log.Println("Could not fetch ", r.URL, ": ", err)
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	defer resp.Body.Close()
    
    //写入相关得响应头、状态码、实体数据
    //实现转发代理功能
	copyHeader(resp.Header, w)
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

//得到转发的HTTP头部
func copyHeader(from http.Header, to http.ResponseWriter) {
	toHeader := to.Header()
	for k, vs := range from {
		for _, v := range vs {
			toHeader.Add(k, v)
		}
	}
}

//得到需要代理访问的url
func calculateURL(r *http.Request) *url.URL {
	newURL := *r.URL
	oldPath := r.URL.Path
	oldPathParts := strings.Split(oldPath, "/")[1:]
	log.Println("This is oldPath:", oldPath, strings.Split(oldPath, "/"), oldPathParts, len(oldPathParts))
	//之前的代码没有这个判断，但是在浏览器访问时会有数组溢出报错信息
	//排错得知是在字符串切割后并没有进行数组长度的判断，会使程序异常
	if len(oldPathParts) <= 1 {
		log.Println("The URL is too short ")
		newURL.Host = ""
		newURL.Path = ""
		newURL.Scheme = ""
		return &newURL
	}
	newURL.Host = oldPathParts[0] + ":" + oldPathParts[1]
	newURL.Path = "/" + strings.Join(oldPathParts[2:], "/")

	newURL.Scheme = "http"

	return &newURL
}

func main() {
	//定义一个命令行flag
	//flag 名称:port 默认值：8888，提示信息：
	var port = flag.Int("port", 8888, "port that micro-bastion should listen on")
	flag.Parse()

	//打印提示信息
	log.Println("Starting micro-bastion on port", *port)

	//注册http server服务
	server := &http.Server{
		//监听的host：port
		Addr: fmt.Sprint(":", *port),
		//进行强制类型转换，将handleRequest转换为HandlerFunc类型，进行路由注册
		Handler:           http.HandlerFunc(handleRequest),
		ReadTimeout:       0,
		ReadHeaderTimeout: 0,
		WriteTimeout:      0,
		IdleTimeout:       0,
	}

	// start the server
	log.Fatal(server.ListenAndServe())
}
