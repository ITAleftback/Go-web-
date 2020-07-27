package jxp

//中间件最重要的一点是把函数转成HandlerFunc，然后在中间件里调用ServeHTTP
//中间件，将心的中间件加入到了group的handler的slice中
func (group *RouterGroup) Use(middlewares ...HandlerFunc) {
	group.Handlers = append(group.Handlers, middlewares...)
}

