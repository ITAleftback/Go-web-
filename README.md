
# httprouter

看完有关介绍，发现诸多的web框架都使用这个httprouter来弥补部分net/http的不足。看了下原理，究其原因其实就是使用了一个前缀树来管理请求的URL。

httprouter中, 对于每种方法都有一颗tree来管理, 例如所有的GET方法对应的请求会有一颗tree管理, 所有的POST同样如此. 先来看看router

```
type Router struct {
   trees map[string]*node
   
   RedirectTrailingSlash bool
   
   RedirectFixedPath bool
   
   HandleMethodNotAllowed bool
  
   HandleOPTIONS bool

   GlobalOPTIONS http.Handler

   globalAllowed string

   NotFound http.Handler

   MethodNotAllowed http.Handler

   PanicHandler func(http.ResponseWriter, *http.Request, interface{})
}
```

上面的结构中, trees map[string]*node代表的一个森林, 里面有一颗GET tree, POST tree… 

而我原本想到的是用map嵌套使得method，uri，handleFunc都指定起来，光是这样肯定是太简单了。

再来看看node

```
type node struct {
   // 保存这个节点上的URL路径
   path      string
  
   wildChild bool
   nType     nodeType
   maxParams uint8
   priority  uint32
   indices   string
   children  []*node
   //一种请求函数
   handle    Handle
}
```

学识浅陋的我只能看懂了path与handle。如上注释所示。



于是我大概似懂非懂的懂了过程。用map保存method，而每个method对应一个树，这个树上有路径及其对应的handle。

**所以说框架会生成几棵基数树，分别对应http method。树结构对应path的树状形态，每个path会map到一个HandlerFunc。这样，在接收到http请求的时候，先找到该请求对应的http method树，再根据URL的path找到注册好的HandlerFunc。HandlerFunc对应的业务逻辑需要gin的使用者借助Context结构自行处理。**



那分组路由应该怎么实现呢？我又跑去看了看gin的源码，发现里面有个RouterGroup与Engine互相继承。而RouterGroup顾名思义则是拿来存放的地方。于是我想到了，用RouterGroup来存路径。如果是直接实现路由，则直接向prefix添加路径。如果是分组路由，则可以先保存前面的路径，然后后面再通过某个函数讲前面的路径与后面注册的路径进行整合，在将其赋给prefix。



# 实现

## 路由

```
//开路由组的结构体
type RouterGroup struct {
   Handlers HandlerFunc//存放接口函数
   prefix string//存放路径

   engine *Engine
}


type Engine struct {
   //继承上面的结构体方便用一些东西
   *RouterGroup
   router *httprouter.Router
}
```

Engine里的router，我原本想到的是用map嵌套实现，两层嵌套，method、路径以及接口函数对应起来。

而我尝试使用了下httprouter来直接注册路由。

```
a:=httprouter.New()
a.GET("/test",hello)
```

发现是可行的，也就是说，我可以直接使用httprouter里的router结构体。



然后是定义的Context数据，相关注释已写。

```
//Context
type Context struct {
   Req *http.Request//发送请求
   Writer http.ResponseWriter//服务器响应
   Params httprouter.Params//参数放这
   //Param map[string]string//
   
   handlers HandlerFunc//
   engine *Engine
   index int8//用来遍历 切片
   Errors   []ErrorMsg//错误信息
}

type HandlerFunc func(*Context)
```

然后从最简单的开始，Defalut。

```
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
```

defalut在于将Engine初始化然后返回，方便后面的一些如写中间件的操作。值得一提的是httprouter.new，其返回的正是Router结构体，如此一来我就可以实现上面说的前缀树管理URL。



然后是最简单的的启动函数，就是把监听函数简简单单的封装了一下。

```
//先把最简单的写了，启动函数，只是把http.ListenAndServe封装一下
func (engine *Engine)Run(addr string){
   _ = http.ListenAndServe(addr, engine)
}
```

但是这里曾遇到过一个小问题。engine传入这个参数时是会报错的，我很纳闷去查看了监听函数的源码，发现第二个参数的形式是一个接口

```
type Handler interface {
   ServeHTTP(ResponseWriter, *Request)
}
```

里面有个方法ServeHTTP，所以只要engine实现这个方法，就可以当参数了

```
func (engine *Engine) ServeHTTP(w http.ResponseWriter, req *http.Request) {
   engine.router.ServeHTTP(w, req)
}
```

那么，这个函数又应该拿来干嘛的？我想了下应该是将先前注册的路由实现接口，ServeHTTP可以将engine传给ListenAndServe，然后中间所有的路由管理，path对应的handler等，都由ServeHTTP去调用相应的实现函数

而我为了方便就直接调用了httprouter里的ServeHTTP

下面是httprouter里的ServeHTTP代码

```
// ServeHTTP makes the router implement the http.Handler interface.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
   if r.PanicHandler != nil {
      defer r.recv(w, req)
   }

   path := req.URL.Path

   if root := r.trees[req.Method]; root != nil {
      if handle, ps, tsr := root.getValue(path); handle != nil {
         handle(w, req, ps)
         return
      } else if req.Method != http.MethodConnect && path != "/" {
         code := 301 // Permanent redirect, request with GET method
         if req.Method != http.MethodGet {
            // Temporary redirect, request with same method
            // As of Go 1.3, Go does not support status code 308.
            code = 307
         }

         if tsr && r.RedirectTrailingSlash {
            if len(path) > 1 && path[len(path)-1] == '/' {
               req.URL.Path = path[:len(path)-1]
            } else {
               req.URL.Path = path + "/"
            }
            http.Redirect(w, req, req.URL.String(), code)
            return
         }

         // Try to fix the request path
         if r.RedirectFixedPath {
            fixedPath, found := root.findCaseInsensitivePath(
               CleanPath(path),
               r.RedirectTrailingSlash,
            )
            if found {
               req.URL.Path = string(fixedPath)
               http.Redirect(w, req, req.URL.String(), code)
               return
            }
         }
      }
   }

   if req.Method == http.MethodOptions && r.HandleOPTIONS {
      // Handle OPTIONS requests
      if allow := r.allowed(path, http.MethodOptions); allow != "" {
         w.Header().Set("Allow", allow)
         if r.GlobalOPTIONS != nil {
            r.GlobalOPTIONS.ServeHTTP(w, req)
         }
         return
      }
   } else if r.HandleMethodNotAllowed { // Handle 405
      if allow := r.allowed(path, req.Method); allow != "" {
         w.Header().Set("Allow", allow)
         if r.MethodNotAllowed != nil {
            r.MethodNotAllowed.ServeHTTP(w, req)
         } else {
            http.Error(w,
               http.StatusText(http.StatusMethodNotAllowed),
               http.StatusMethodNotAllowed,
            )
         }
         return
      }
   }

   // Handle 404
   if r.NotFound != nil {
      r.NotFound.ServeHTTP(w, req)
   } else {
      http.NotFound(w, req)
   }
}
```

看着挺花眼的，但仔细一看最重要的在于

```
path := req.URL.Path

if root := r.trees[req.Method]; root != nil {
   if handle, ps, tsr := root.getValue(path); handle != nil {
      handle(w, req, ps)
      return
```

首先拿出http请求里面的method和path路径，然后利用getValue函数返回一个给定路径的handler也就是实现接口传给ListenAndServe。

如此一来，ServeHTTP完成了他的功能，函数可以开始监听。



然后是分组路由

```
// 如果是开路由组，则用到这个方法
func (group *RouterGroup) Group(component string) *RouterGroup {


   return &RouterGroup{
      Handlers: nil,
      prefix:  component,
      engine:  group.engine,
   }
}
```

分组路由与普通的路由有什么区别呢？无非不过是需要将前缀的路径与后面加入的路径合并起来进行注册。所以为此我写了RouterGroup的结构体，prefix就是存放路径，Handlers则是为了放置中间件及其接口函数

而Group这个方法的作用就显而易见了，将路径保存到prefix，并返回一个新的RouterGroup。



然后是写handle，handle的作用是将method ，uri 以及中间件还有接口函数对应起来然后加入到router的路由配置中

```
func (group *RouterGroup) Handle(method, uri string, handler HandlerFunc) {
   //加上原本路由组的 路径
   uris:= path.Join(group.prefix, uri)
 
   //将 整合的路径 还有 handler 添加到httprouter去
   group.engine.router.Handle(method, uris, func(w http.ResponseWriter, req *http.Request, params httprouter.Params) {
      group.NewContext(w, req, params, handler).Next()
   })
}
```

显然，uris正是将路由组的路径与新注册的路径进行整合后的产物。



group.engine.router.Handle是干嘛的？上文提到过，Handle函数作用是将method，uri以及对应的接口函数对应起来，一开始我尝试用map的方式将其对应，但是当加入中间件的时候，对应的又不只有接口函数，不知怎么办的我又去找我的金钥匙了。我发现里面还有一个Handle，所以我又是调用了httprouter里的Handle。



httprouter里Handle的源码

```
func (r *Router) Handle(method, path string, handle Handle) {
   if len(path) < 1 || path[0] != '/' {
      panic("path must begin with '/' in path '" + path + "'")
   }

   if r.trees == nil {
      r.trees = make(map[string]*node)
   }

   root := r.trees[method]
   if root == nil {
      root = new(node)
      r.trees[method] = root

      r.globalAllowed = r.allowed("*", "")
   }

   root.addRoute(path, handle)
}
```

显而易见，最重要的代码块是

```

root := r.trees[method]
if root == nil {
   root = new(node)
   r.trees[method] = root

   r.globalAllowed = r.allowed("*", "")
}

root.addRoute(path, handle)
```

这里就是将传入的method以及path以及handle对应。先是利用map保存方法，然后在利用addRoute函数进行对应。





```
group.engine.router.Handle(method, uris, func(w http.ResponseWriter, req *http.Request, params httprouter.Params) {
   group.NewContext(w, req, params, handlers).Next()
})
```

所以在这个代码块里，传入的正应该是method，uri，以及接口函数。而第三个参数形式为

```
type Handle func(http.ResponseWriter, *http.Request, Params)
```

Handle是一个可以注册到路由以处理HTTP请求的函数

所以我传入第三个匿名函数同时创建了Newcontext。

```
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
```

可能你会注意到Newcontext后面还有一个Next。

这个函数正是我前面提到的调用handler的函数，上源码。

```
func (c *Context) Next() {
	c.handler(c)
}
```



## 中间件

下面讲讲中间件的实现。

先上代码。

```
//中间件最重要的一点是把函数转成HandlerFunc，然后在中间件里调用ServeHTTP
//中间件，将心的中间件加入到了group的handler的slice中
func (group *RouterGroup) Use(middlewares ...HandlerFunc) {
   group.Handlers = append(group.Handlers, middlewares...)
}
```

很简单，只是将传入任意的handlefunc进行整合然后返回到Routergroup里面的handle切片。



其实中间件的原理在于，它是同接口函数一样的函数，但如果需要在接口函数之前实现。很容易就想到切片，先加入的放前面，后加入的放后面，然后再依次调用，如此达到效果。

所以要改的地方就有很多了。

因为是要用到切片，存放HandlerFunc改为切片，用来放置中间件

```
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
```

Handle函数里需要重新定义。

```
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
```

不只是要把HandlerFunc设置成切片形式，还需要考虑到将先去的与后加入的handle进行整合。

所以CombineHandlers函数应运而生。



CombineHandlers代码如下

```
func (group *RouterGroup) CombineHandlers(handlers []HandlerFunc) []HandlerFunc {
	lens := len(group.Handlers) + len(handlers)
	handlerss := make([]HandlerFunc,0, lens)
	handlerss = append(handlerss, group.Handlers...)
	handlerss = append(handlerss, handlers...)
	return handlerss
}
```

显而易见，该函数作用就是将当前的Handler和新来的handler整合，并以切片的形式返回。因为后面有Next函数会帮助进行遍历这个切片，一般来说除了最后一个函数其余的都是中间件。



上面说到了Next函数会帮助进行遍历切片，因为现在HandlerFunc是个切片了，里面有多个函数，需要依次遍历并拿出来调用，且需要注意的是一定要按顺序遍历，如此才能达到中间件效果。

话不多说，上代码。

```
//这个时候需要将silce中的hander依次取出来调用
func (c *Context) Next() {
   c.index++
   //   一般来说 除了最后一个函数  前面的都是中间件
   s := int8(len(c.handlers))
   for ; c.index < s; c.index++ {
      c.handlers[c.index](c)
   }

}
```

现在，Context里的index也可以解释了，为什么Newcontext会设置-1也能讲。

```
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
```

Context里的index就是拿来帮助Next进行遍历切片的，每次有一个GET或POST等的路由注册访问时，都会调用NewContext，至于为什么是-1，很简单，Next函数里会有index++。



## 功能

接下来就是一些框架功能，如我在以前用中间件的时候用set以及get函数获取中间变量。

```
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
```

还是利用了在Context里面的keys数据类型为map保存。在Set函数里就是把Context里的keys用map对应起来，因为value可以是任何数据所以定义的类型为接口。

所以向Contest添加数据

```
Keys map[string]interface{}//用map存储中间变量

```

然后是Get函数，需要把前面在Context保存的变量拿出来，当然key必须是已经保存了的，如果为空说明没有变量存在。



```
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

```

以上是我封装的一些功能。嗯，差不多都是抄的:joy:







## Recovery

其实我并没有敲出recovery的代码....，但是我在网上看到说recovery又是必不可少的异常捕捉，所以我跑去抄了一段。

```
var (
   dunno     = []byte("???")
   centerDot = []byte("·")
   dot       = []byte(".")
   slash     = []byte("/")
)



//func Recovery() HandlerFunc {
// return func(c *Context) {
//    defer func() {
//       if err := recover(); err != nil {
//          message := fmt.Sprint("%s", err)
//          log.Printf("%s\n\n", trace(message))
//          c.Fail(http.StatusInternalServerError, "Interanl server Error")
//       }
//    }()
//
//    c.Next()
// }
//}
//
//func trace(message string) interface{} {
// var pcs [32]uintptr
// n := runtime.Callers(3, pcs[:])
//
// var str strings.Builder
// str.WriteString(message + "\nTraceback:")
// for _, pc := range pcs[:n] {
//    fn := runtime.FuncForPC(pc)
//    file, line := fn.FileLine(pc)
//    str.WriteString(fmt.Sprintf("\n\t%s:%d", file, line))
// }
//
// return str.String()
//}


// stack returns a nicely formated stack frame, skipping skip frames
func stack(skip int) []byte {
   buf := new(bytes.Buffer) // the returned data
   // As we loop, we open files and read them. These variables record the currently
   // loaded file.
   var lines [][]byte
   var lastFile string
   for i := skip; ; i++ { // Skip the expected number of frames
      pc, file, line, ok := runtime.Caller(i)
      if !ok {
         break
      }
      // Print this much at least.  If we can't find the source, it won't show.
      fmt.Fprintf(buf, "%s:%d (0x%x)\n", file, line, pc)
      if file != lastFile {
         data, err := ioutil.ReadFile(file)
         if err != nil {
            continue
         }
         lines = bytes.Split(data, []byte{'\n'})
         lastFile = file
      }
      fmt.Fprintf(buf, "\t%s: %s\n", function(pc), source(lines, line))
   }
   return buf.Bytes()
}

func source(lines [][]byte, n int) []byte {
   n--
   if n < 0 || n >= len(lines) {
      return dunno
   }
   return bytes.TrimSpace(lines[n])
}


func function(pc uintptr) []byte {
   fn := runtime.FuncForPC(pc)
   if fn == nil {
      return dunno
   }
   name := []byte(fn.Name())

   if lastslash := bytes.LastIndex(name, slash); lastslash >= 0 {
      name = name[lastslash+1:]
   }
   if period := bytes.Index(name, dot); period >= 0 {
      name = name[period+1:]
   }
   name = bytes.Replace(name, centerDot, dot, -1)
   return name
}

//只有在延迟函数内部调用Recover才有用。
// 在延迟函数内调用 recover， 可以取到 panic 的错误信息，
// 并且停止 panic 续发，程序运行恢复正常
func Recovery() HandlerFunc {
   return func(c *Context) {
      defer func() {
         if len(c.Errors) > 0 {
            log.Println(c.Errors)
         }
         if err := recover(); err != nil {
            stack := stack(3)
            log.Printf("PANIC: %s\n%s", err, stack)
            c.Writer.WriteHeader(http.StatusInternalServerError)
         }
      }()

      c.Next()
   }
}
```



# 遇到的问题

## 1.GET与POST是如何实现区别

POST比GET的安全性更高，那我该怎样写实现这个”安全“？在写原生的过程中，我遇到了这个问题。不管是看了gin的源码还是httprouter的源码，都没有找到相关的解决方法，源码里好像从来没有特别的对method进行解析，也没有分析格式。到后来与朋友的交谈才得知，这样的区别是浏览器自动实现。

![](C:\Users\Mechrevo\Pictures\QQ图片20200728012130.png)

也就是说，只需要后端上传的method是GET或者是POST，浏览器就会自动解析，然后打包上传，是浏览器的内核在运行。



## 2.结构体的互相调用

在查看gin的源码，RouterGroup与Engine互相是对方的成员。于是我有一个疑惑，如此的话，便会无限分配内存下去，肯定是不可能的。

```
type B struct {
   *A
}

type A struct {
   *B
}
```

与朋友交谈后得以解决。

如果是指针，a结构体内写了去找b结构体的地址，b结构体内写了去找a的地址，在调用a的时候，编译器会给B分配内存，但不会给B的成员变量分配内存，就不会循环。所以，如果不是指针的话，编译器会自动报错。



