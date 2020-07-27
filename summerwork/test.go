package main

import (
	"fmt"
	"strings"
)

func main() {
	url:="http://localhost:8080/cake?a=1&b=2"
	uris :=strings.Split(url,"?")
	fmt.Println(uris)
	fmt.Println(len(uris))
	//
	if len(uris)==1{
		return
	}
	param:=uris[len(uris)-1]
	fmt.Println(param)
	//如果有参数
	pair :=strings.Split(param,"&")
	fmt.Println(pair)
}
