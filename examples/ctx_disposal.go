// Added to reproduce and test issue 48 (missing Dispose causing leak)
// https://github.com/dss-extensions/dss-extensions/issues/48
package main

import (
	"log"
	"runtime"
	"time"
	"github.com/dss-extensions/altdss-go/altdss"
)

func main() {
	dss := altdss.IDSS{}
	dss.Init(nil)
	for i := 0; i < 10000; i++ {
		time.Sleep(1 * time.Millisecond)
		ctx, err := dss.NewContext()
		ctx.Dispose()
		if err != nil {
			log.Fatal(err)
		}
		runtime.GC()
	}
}
