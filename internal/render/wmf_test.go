package render

import (
	"encoding/binary"
	"testing"
)

func placeableWMFForTest(left, top, right, bottom int16) []byte {
	b := make([]byte, 22)
	binary.LittleEndian.PutUint32(b[0:4], PlaceableWMFKey)
	binary.LittleEndian.PutUint16(b[6:8], uint16(left))
	binary.LittleEndian.PutUint16(b[8:10], uint16(top))
	binary.LittleEndian.PutUint16(b[10:12], uint16(right))
	binary.LittleEndian.PutUint16(b[12:14], uint16(bottom))
	return b
}

func TestParseWMFBoundsPlaceable(t *testing.T) {
	data := append(placeableWMFForTest(10, 20, 1010, 1020), 0x01, 0x02, 0x03)
	metaX, metaY, metaW, metaH, metaData, err := ParseWMFBounds(data)
	if err != nil {
		t.Fatal(err)
	}
	if metaX != 10 || metaY != 20 || metaW != 1000 || metaH != 1000 || len(metaData) != 3 {
		t.Fatalf("bounds=(%d,%d,%d,%d) data=%#v", metaX, metaY, metaW, metaH, metaData)
	}
}

func TestParseWMFBoundsNonPlaceable(t *testing.T) {
	data := make([]byte, 22)
	data[0] = 0x01
	metaX, metaY, metaW, metaH, metaData, err := ParseWMFBounds(data)
	if err != nil {
		t.Fatal(err)
	}
	if metaX != 0 || metaY != 0 || metaW != 1000 || metaH != 1000 || len(metaData) != len(data) {
		t.Fatalf("non-placeable bounds=(%d,%d,%d,%d) data=%d", metaX, metaY, metaW, metaH, len(metaData))
	}
}

func TestParseWMFBoundsRejectsInvalid(t *testing.T) {
	if _, _, _, _, _, err := ParseWMFBounds([]byte{1, 2}); err == nil {
		t.Fatal("expected error for short WMF")
	}
	if _, _, _, _, _, err := ParseWMFBounds(placeableWMFForTest(10, 20, 10, 30)); err == nil {
		t.Fatal("expected error for zero-width placeable WMF")
	}
}
