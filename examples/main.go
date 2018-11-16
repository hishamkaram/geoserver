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
	groups, groupsErr := gsCatalog.GetLayerGroups("geonode")
	if groupsErr != nil {
		panic("error")
	}
	for _, group := range groups {
		fmt.Printf("Name: %s\t Location: %s.\n", group.Name, group.Href)
	}
	group, groupErr := gsCatalog.GetLayerGroup("geonode", "test")
	if groupErr != nil {
		panic("error")
	}

	fmt.Printf("Group:%+v,\n>>>>>%+v", group, group.Publishables.Published[0])
	// w := geoserver.Resource{Name: "geonode"}
	var y interface{}
	y = geoserver.BoundingBox{
		Minx: -74.2555923461914,
		Maxx: -73.7000045776367,
		Miny: 40.4961128234863,
		Maxy: 40.9155349731445}

	switch y.(type) {
	case interface{}:
		fmt.Println("InterFace")
	}
	// c := geoserver.LayerGroup{Name: "golang",
	// 	Title:     "Go",
	// 	Mode:      "SINGLE",
	// 	Workspace: &w,
	// 	Publishables: geoserver.Publishables{Published: []*geoserver.GroupPublishableItem{
	// 		{Name: "nyc_fatality_neighbourhood_2a3e3916", Href: "http://localhost/geoserver/rest/layers/nyc_fatality_neighbourhood_2a3e3916.json"},
	// 	}}, Styles: geoserver.LayerGroupStyles{Style: []*geoserver.Resource{
	// 		{Name: "nyc_fatality_neighbourhood_2a3e3916", Href: "http://localhost/geoserver/rest/workspaces/geonode/styles/nyc_fatality_neighbourhood_2a3e3916.json"},
	// 	}}, Bounds: geoserver.NativeBoundingBox{
	// 		BoundingBox: geoserver.BoundingBox{
	// 			Minx: -74.2555923461914,
	// 			Maxx: -73.7000045776367,
	// 			Miny: 40.4961128234863,
	// 			Maxy: 40.9155349731445},
	// 		Crs: &x,
	// 	}}
	// created, createErr := gsCatalog.CreateLayerGroup("geonode", &c)
	// if createErr != nil {
	// 	panic(createErr)
	// }
	// fmt.Printf("layerGroup Created")
	// fmt.Println(created)

}
