package main

import (
	"fmt"

	"github.com/hishamkaram/geoserver"
)

func main() {
	gsCatalog := geoserver.GetCatalog("http://localhost:8080/geoserver", "admin", "geoserver")
	cap, err := gsCatalog.GetCapabilities("")
	if err != nil {
		panic("error")
	}
	for _, srs := range cap.Capability.Layer.SRS {
		fmt.Printf("%v\n", *srs)
		fmt.Println("+++++++++++++++++++++++++")
	}
}
