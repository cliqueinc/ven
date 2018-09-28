package main

import (
	"os"
	"runtime"

	"github.com/gohugoio/hugo/commands"
	jww "github.com/spf13/jwalterweatherman"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	commands.Execute()

	if jww.LogCountForLevelsGreaterThanorEqualTo(jww.LevelError) > 0 {
		os.Exit(-1)
	}

	if commands.Hugo != nil {
		if commands.Hugo.Log.LogCountForLevelsGreaterThanorEqualTo(jww.LevelError) > 0 {
			os.Exit(-1)
		}
	}
}
