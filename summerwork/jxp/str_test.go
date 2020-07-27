package jxp

import (
	"fmt"
	"testing"
)


func Test(t *testing.T){
	var B B
	B.HAHA()
}

type B struct {
	*B
}

func (b *B)HAHA(){
	fmt.Println(111)
}
