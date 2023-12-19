package main

import (
	"downloader/service"
	"fmt"
)

func main() {
	fmt.Println("Downloader Started")
	service.Server()
}
