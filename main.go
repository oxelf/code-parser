package main

import "code-parser/server"

func main() {
	err := server.StartServer()
	if err != nil {
		panic(err)
	}
}
