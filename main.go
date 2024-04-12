package main

import "os"

func main() {
	// print the message: "its working"
	println("Worker its working")
	// gracefully shutdown this program
	os.Exit(0)
}
