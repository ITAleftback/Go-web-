package jxp

import (
	"encoding/json"
	"github.com/julienschmidt/httprouter"
	"log"
	"net/http"
)





type ErrorMsg struct {
	Message string      `json:"msg"`
	Meta    interface{} `json:"meta"`
}

type H map[string]interface{}

//Context
type Context struct {
	Req *http.Request//发送请求
	Writer http.ResponseWriter//服务器响应
	Params httprouter.Params//参数放这
	//Param map[string]string//拿参数
	Keys map[string]interface{}//用map存储中间变量
	handlers []HandlerFunc//
	engine *Engine
	index int8//用来遍历 切片
	Errors   []ErrorMsg//错误信息
}

type HandlerFunc func(*Context)


//
func (c *Context) String(code int, msg string) {
	c.Writer.Header().Set("Content-Type", "text/plain")
	c.Writer.WriteHeader(code)//把状态码发送到请求头?
	//在网站上打印
	_, _ = c.Writer.Write([]byte(msg))
}
//这个JSON我自己没写出来，是跟网上写的，看了下，是请求头有关
func (c *Context) JSON(code int, obj interface{}) {
	c.Writer.Header().Set("Content-Type", "application/json")
	c.Writer.WriteHeader(code)

	encoder := json.NewEncoder(c.Writer)
	if err := encoder.Encode(obj); err != nil {
		http.Error(c.Writer, err.Error(), 500)
	}
}

//这个时候需要将silce中的hander依次取出来调用
func (c *Context) Next() {
	c.index++
	//   一般来说 除了最后一个函数  前面的都是中间件
	s := int8(len(c.handlers))
	for ; c.index < s; c.index++ {
		c.handlers[c.index](c)
	}

}

//拿参数
func (c *Context) PostForm(key string) string {
	return c.Req.FormValue(key)
}
//封装的一些功能
func (c *Context) Query(key string) string {
	return c.Req.URL.Query().Get(key)
}

func (c *Context) SetHeader(key string, value string) {
	c.Writer.Header().Set(key, value)
}

func (c *Context) Data(code int, data []byte) {
	_, _ = c.Writer.Write(data)
}

func (c *Context) HTML(code int, html string) {
	c.SetHeader("Content-Type", "text/html")
	_, _ = c.Writer.Write([]byte(html))
}





// 存储中间变量
func (c *Context) Set(key string, item interface{}) {
	if c.Keys == nil {
		c.Keys = make(map[string]interface{})
	}
	c.Keys[key] = item
}


//读取存储的变量
func (c *Context)Get(key string)interface{}{
	var ok bool
	var item interface{}
	//如果  是
	if c.Keys != nil {
		item, ok = c.Keys[key]
	} else {
		item, ok = nil, false
	}
	// item为空 或者 ok为false的话说明就没有变量存在
	if !ok || item == nil {
		log.Panicf("Key %s doesn't exist", key)
	}
	return item
}






