package main

import (
	"log"

	"github.com/Dids/go-broadlink/broadlink"
)

func main() {
	device := broadlink.New()
	log.Println("Device discovered:", device.IP)
	log.Println("Exiting..")
}
