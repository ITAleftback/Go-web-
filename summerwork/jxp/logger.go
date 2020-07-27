package jxp

import (
	"log"
	"time"
)

func Logger()HandlerFunc{
	return func(c *Context) {

		t:=time.Now()

		c.Next()

		log.Printf("%s in %v",c.Req.RequestURI,time.Since(t))
	}
}
