package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/linyuanbin/viper-reload/src/pkg/cfg"
)

func main() {
	config := cfg.NewConfig()
	if err := config.Parse(); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(config)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	//开启监听
	config.WatchConfig(ctx)
	http.HandleFunc("/", sayhelloName)       //设置访问的路由
	err := http.ListenAndServe(":9090", nil) //设置监听的端口
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func sayhelloName(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	fmt.Println(r.Form)
	fmt.Println("path", r.URL.Path)
	fmt.Println("scheme", r.URL.Scheme)
	fmt.Println(r.Form["url_long"])
	for k, v := range r.Form {
		fmt.Println("key:", k)
		fmt.Println("val:", strings.Join(v, ""))
	}
	fmt.Fprintf(w, "Hello astaxie!")
}
