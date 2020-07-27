package jxp

import (
	"github.com/julienschmidt/httprouter"
	"net/http"
	"path"
)
/*
尝试用原生写时，  想到的是用map嵌套使得method指向路径再指向HandleFunc
但是在尝试写路由组的时候遇到了问题，
路由组的话需要将路径整合，但是在Handle函数里 将整合的路径传入不知是什么错

于是去网上找解决方法 发现了httprouter这个金钥匙


 */



//type HandlerFunc func(*Context)
//
//type handlerMaps map[string]HandlerFunc

//type Engine struct {
//
//	router map[string]handlerMaps//map嵌套  分别是method uri 以及对应的函数
//}
//


//开路由组的结构体
type RouterGroup struct {
	Handlers []HandlerFunc//设置成切片是为了  存放中间件
	prefix string

	engine *Engine
}


type Engine struct {
	//继承上面的结构体方便用一些东西
	*RouterGroup
	router *httprouter.Router
}


// 类似与 router:=gin.Default 初始化
func Defalut() *Engine{
	engine:=&Engine{}
	engine.RouterGroup=&RouterGroup{
		Handlers: nil,
		prefix:  "",

		engine:  engine,
	}
	//用这个生成*router 路由指针
	engine.router=httprouter.New()
	return engine
}


//先把最简单的写了，启动函数，只是把http.ListenAndServe封装一下
func (engine *Engine)Run(addr string){
	_ = http.ListenAndServe(addr, engine)
}
//我发现 启动函数里面的ListenAndServe第二个参数会报错 看了下是接口，里面有个方法ServeHTTP
// 所以只要engine实现 这个方法就可以当参数了
func (engine *Engine) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	engine.router.ServeHTTP(w, req)
}

// 如果是开路由组，则用到这个方法
func (group *RouterGroup) Group(component string) *RouterGroup {


	return &RouterGroup{
		Handlers: nil,
		prefix:  component,
		engine:  group.engine,
	}
}


////判断是否有参数 并取出
//func ParseQuery(uri string)(res map[string]string){
//	res=make(map[string]string)
//	uris:=strings.Split(uri,"?")
//	if len(uris)==1 {
//		//如果等于1 说明没有参数 返回即可
//		return
//	}
//	param:=uris[len(uris)-1]//用来拿后面的参数
//	pair:=strings.Split(param,"&") //将多个参数分割  并遍历
//
//	for _,v:=range pair{
//		kvpair:=strings.Split(v,"=")//前面的是key 后面的是参数
//		res[kvpair[0]]=kvpair[1]//用map保存
//	}
//	return
//}


func (group *RouterGroup) Handle(method, uri string, handlers []HandlerFunc) {
	//加上原本路由组的 路径
	uris:= path.Join(group.prefix, uri)
	//将新来的handler整合
	handlers = group.CombineHandlers(handlers)
	//将 整合的路径 还有 handler 添加到httprouter去
	group.engine.router.Handle(method, uris, func(w http.ResponseWriter, req *http.Request, params httprouter.Params) {
		group.NewContext(w, req, params, handlers).Next()
	})
}


//给前面的Context赋值
func (group *RouterGroup) NewContext(w http.ResponseWriter, req *http.Request, params httprouter.Params, handlers []HandlerFunc) *Context {
	return &Context{
		Writer:  w,
		Req:     req,
		index:   -1,// 第一次就设置-1  调用时同时会调用next++
		engine:  group.engine,
		Params:  params,
		handlers: handlers,
	}
}
//

//// 有关解析的调用
//func (engine *Engine)handle(method,uri string,handler HandlerFunc)  {
//	handlers, ok := engine.router[method]
//	if !ok {
//		m := make(handlerMaps)
//		engine.router[method] = m
//		handlers = m
//	}
//	_, ok = handlers[uri]
//	if ok {
//		panic("same route")
//	}
//
//	handlers[uri] = handler
//}



//这个是将多个handler聚合，然后一块传入到router，作用就是将当前的Handler和新来的handler整合
func (group *RouterGroup) CombineHandlers(handlers []HandlerFunc) []HandlerFunc {
	lens := len(group.Handlers) + len(handlers)
	h := make([]HandlerFunc,0, lens)
	h = append(h, group.Handlers...)
	h = append(h, handlers...)
	return h
}



//==========================请求头==========================

// 请求头的作用还有是 将前面的路径整合起来  然后将对应的路径和 handler加入到httprouter
func (group *RouterGroup) POST(path string, handlers ...HandlerFunc) {
	group.Handle("POST", path, handlers)
}

//
func (group *RouterGroup) GET(path string, handlers ...HandlerFunc) {
	group.Handle("GET", path, handlers)
}

func (group *RouterGroup) DELETE(path string, handlers ...HandlerFunc) {
	group.Handle("DELETE", path, handlers)
}


func (group *RouterGroup) PATCH(path string, handlers ...HandlerFunc) {
	group.Handle("PATCH", path, handlers)
}


func (group *RouterGroup) PUT(path string, handlers ...HandlerFunc) {
	group.Handle("PUT", path, handlers)
}
