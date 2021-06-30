package main

import (
	"fmt"

	"github.com/IDN-Media/awards/internal"
	log "github.com/sirupsen/logrus"
)

var (
	splashScreen = ` 
    __ ___      ____ _ _ __ __| |___ 
   / _  \ \ /\ / / _  |  __/ _  / __|
  | (_| |\ V  V / (_| | | | (_| \__ \
   \__,_| \_/\_/ \__,_|_|  \__,_|___/
 				 

	(c)2021 idn-media awards server
	https://github.com/idn-media/awards/README.md  
	
	`
)

func init() {
	fmt.Println(splashScreen)
	log.Info("initialzing...")
}

// Main entry point
func main() {

	// start server
	internal.StartServer()
}
