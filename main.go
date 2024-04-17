package main

import (
	// "os"
	// "runtime/pprof"

	"github.com/aayustark007/wcgo/cmd"
)

func main() {
	// f, err := os.Create("wcgo.prof")
	// if err != nil {
	// 	panic(err)
	// }
	// defer f.Close()
	// pprof.StartCPUProfile(f)
	// defer pprof.StopCPUProfile()

	// runtime.GC()
	cmd.Execute()

	// if err := pprof.WriteHeapProfile(f); err != nil {
	// 	panic(err)
	// }
}
