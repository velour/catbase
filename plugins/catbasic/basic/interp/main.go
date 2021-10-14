package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/velour/catbase/plugins/catbasic/basic"
	"github.com/velour/catbase/plugins/catbasic/basic/lang"
)

const source = `
FOR I = 0 TO 5: PRINT "I:", I: NEXT I
`

func main() {
	interp, err := basic.New(lang.Default)
	if err != nil {
		die(err)
	}
	if err := interp.Read(strings.NewReader(source)); err != nil {
		die(err)
	}
}

func die(err error) {
	fmt.Println(err.Error())
	os.Exit(1)
}
