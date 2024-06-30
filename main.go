package main

import (
	"SplitFlapApp/proto"
	"SplitFlapApp/restfulApi"
)

func main() {
	if proto.ShouldGenerateProtoFiles() {
		proto.GenerateProtoFiles()
	}

	restfulManager := restfulApi.NewRestfulManager()

	restfulManager.Start(false)
}
