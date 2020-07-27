package main

import (
	"github.com/julienschmidt/httprouter"
	"summerwork/jxp"
)
func main(){
	r := jxp.Defalut()

	a:=httprouter.New()
	a.GET()
	//开个路由组
	router := r.Group("/jxp")
	{
		router.GET("/test3", hello)
		router.GET("/test2",test2)
		router.GET("/test",jxp.Logger(),test)

	}

	r.Run(":8080")

}
func test2(c *jxp.Context)  {
	c.String(200,"test success")
}

func hello(c *jxp.Context)  {
	name:=c.PostForm("name")
	c.String(200,"hello!"+name)
}

func test(c *jxp.Context)  {
	name:=c.PostForm("name")
	c.Set("innerName",name)
	message:=getInfo(c)

	c.JSON(200,jxp.H{"message":message})

}

func getInfo(c *jxp.Context)string{
	name:=c.Get("innerName")
	message:= "welcome！ "+name.(string)
	return message
}
