//go:build darwin || cgo
// +build darwin cgo

package main

import "C"
import "fmt"

//export StartGoServer
func StartGoServer() {
	fmt.Println("StartGoServer called from macOS dynamic library (FFI)...")
	prg := &program{}
	go prg.run()
}
