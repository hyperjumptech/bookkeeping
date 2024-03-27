package main

import (
	"fmt"

	"github.com/hyperjumptech/bookkeeping/internal"
)

var (
	splashScreen = ` 
██████╗  ██████╗  ██████╗ ██╗  ██╗██╗  ██╗███████╗███████╗██████╗ ██╗███╗   ██╗ ██████╗
██╔══██╗██╔═══██╗██╔═══██╗██║ ██╔╝██║ ██╔╝██╔════╝██╔════╝██╔══██╗██║████╗  ██║ ██╔════╝     
██████╔╝██║   ██║██║   ██║█████╔╝ █████╔╝ █████╗  █████╗  ██████╔╝██║██╔██╗ ██║ ██║  ███╗  
██╔══██╗██║   ██║██║   ██║██╔═██╗ ██╔═██╗ ██╔══╝  ██╔══╝  ██╔═══╝ ██║██║╚██╗██║ ██║   ██║
██████╔╝╚██████╔╝╚██████╔╝██║  ██╗██║  ██╗███████╗███████╗██║     ██║██║ ╚████║ ╚██████╔╝
╚═════╝  ╚═════╝  ╚═════╝ ╚═╝  ╚═╝╚═╝  ╚═╝╚══════╝╚══════╝╚═╝     ╚═╝╚═╝  ╚═══╝  ╚═════╝
                                                           
	(c)2021-2024 hyperjumptech double bookkeeping service
	https://github.com/hyperjumptech/bookkeeping/README.md  
	
	`
)

func init() {
	fmt.Println(splashScreen)
}

// Main entry point
func main() {

	// start server
	internal.StartServer()
}
