package jxp

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"runtime"
)

//异常捕捉 我并没有写出来， 即使是看了源码也还是很疑惑

var (
	dunno     = []byte("???")
	centerDot = []byte("·")
	dot       = []byte(".")
	slash     = []byte("/")
)



//func Recovery() HandlerFunc {
//	return func(c *Context) {
//		defer func() {
//			if err := recover(); err != nil {
//				message := fmt.Sprint("%s", err)
//				log.Printf("%s\n\n", trace(message))
//				c.Fail(http.StatusInternalServerError, "Interanl server Error")
//			}
//		}()
//
//		c.Next()
//	}
//}
//
//func trace(message string) interface{} {
//	var pcs [32]uintptr
//	n := runtime.Callers(3, pcs[:])
//
//	var str strings.Builder
//	str.WriteString(message + "\nTraceback:")
//	for _, pc := range pcs[:n] {
//		fn := runtime.FuncForPC(pc)
//		file, line := fn.FileLine(pc)
//		str.WriteString(fmt.Sprintf("\n\t%s:%d", file, line))
//	}
//
//	return str.String()
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
