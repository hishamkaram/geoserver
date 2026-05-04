package wire_test

import (
	"encoding/json"
	"testing"

	"github.com/hishamkaram/geoserver/v2/internal/wire"
)

func TestCRS_UnmarshalObject(t *testing.T) {
	var c wire.CRS
	if err := json.Unmarshal([]byte(`{"@class":"projected","$":"EPSG:4326"}`), &c); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if c.Class != "projected" || c.Value != "EPSG:4326" {
		t.Fatalf("CRS = %+v", c)
	}
}

func TestCRS_UnmarshalBareString(t *testing.T) {
	var c wire.CRS
	if err := json.Unmarshal([]byte(`"EPSG:4326"`), &c); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if c.Class != "string" || c.Value != "EPSG:4326" {
		t.Fatalf("CRS = %+v", c)
	}
}

func TestCRS_UnmarshalEmpty(t *testing.T) {
	var c wire.CRS
	// An empty object should fail (neither class nor value present).
	err := json.Unmarshal([]byte(`{}`), &c)
	if err == nil {
		t.Fatalf("expected error on empty object, got nil")
	}
}

func TestCRS_MarshalObject(t *testing.T) {
	c := wire.CRS{Class: "projected", Value: "EPSG:4326"}
	got, err := json.Marshal(&c)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(got) != `{"@class":"projected","$":"EPSG:4326"}` {
		t.Fatalf("marshal = %s", got)
	}
}

func TestCRS_MarshalBareString(t *testing.T) {
	c := wire.CRS{Class: "string", Value: "EPSG:4326"}
	got, err := json.Marshal(&c)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(got) != `"EPSG:4326"` {
		t.Fatalf("marshal = %s", got)
	}
}

func TestCRS_MarshalEmpty(t *testing.T) {
	var c wire.CRS
	got, err := json.Marshal(&c)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(got) != `""` {
		t.Fatalf("marshal = %s", got)
	}
}

func TestCRS_MarshalNil(t *testing.T) {
	var c *wire.CRS
	got, err := json.Marshal(c)
	if err != nil {
		t.Fatalf("marshal nil: %v", err)
	}
	// nil pointer marshals via standard encoding/json behavior — null.
	if string(got) != `null` {
		t.Fatalf("marshal nil = %s, want null", got)
	}
}

func TestBoundingBox_RoundTrip(t *testing.T) {
	bbox := wire.NativeBoundingBox{
		BoundingBox: wire.BoundingBox{MinX: -180, MaxX: 180, MinY: -90, MaxY: 90},
		CRS:         &wire.CRS{Class: "projected", Value: "EPSG:4326"},
	}
	data, err := json.Marshal(bbox)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got wire.NativeBoundingBox
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.MinX != -180 || got.MaxY != 90 {
		t.Fatalf("BoundingBox = %+v", got.BoundingBox)
	}
	if got.CRS == nil || got.CRS.Class != "projected" || got.CRS.Value != "EPSG:4326" {
		t.Fatalf("CRS = %+v", got.CRS)
	}
}
