//go:build windows

package print

import (
	"testing"
	"unsafe"
)

func TestApplyLandscapeOrientation(t *testing.T) {
	buf := make([]byte, int(unsafe.Sizeof(devmodeHeader{})))
	applyLandscapeOrientation(buf, 1)
	dm := (*devmodeHeader)(unsafe.Pointer(&buf[0]))
	if dm.DmOrientation != dmOrientLandscape {
		t.Fatalf("orientation=%d want %d", dm.DmOrientation, dmOrientLandscape)
	}
	if dm.DmFields&dmOrientation == 0 {
		t.Fatal("DM_ORIENTATION not set")
	}
}

func TestApplyLandscapeOrientationPortraitNoop(t *testing.T) {
	buf := make([]byte, int(unsafe.Sizeof(devmodeHeader{})))
	applyLandscapeOrientation(buf, 0)
	dm := (*devmodeHeader)(unsafe.Pointer(&buf[0]))
	if dm.DmFields&dmOrientation != 0 || dm.DmOrientation != 0 {
		t.Fatalf("unexpected portrait devmode: fields=%#x orientation=%d", dm.DmFields, dm.DmOrientation)
	}
}

func TestApplyPrinterCopiesForcesSingleDriverCopy(t *testing.T) {
	buf := make([]byte, int(unsafe.Sizeof(devmodeHeader{})))
	applyPrinterCopies(buf, 1)
	dm := (*devmodeHeader)(unsafe.Pointer(&buf[0]))
	if dm.DmFields&dmCopiesField == 0 {
		t.Fatal("DM_COPIES not set")
	}
	if dm.DmCopies != 1 {
		t.Fatalf("copies=%d want 1", dm.DmCopies)
	}
}

func TestApplyPrinterCopiesClampsToOne(t *testing.T) {
	buf := make([]byte, int(unsafe.Sizeof(devmodeHeader{})))
	applyPrinterCopies(buf, 0)
	dm := (*devmodeHeader)(unsafe.Pointer(&buf[0]))
	if dm.DmCopies != 1 {
		t.Fatalf("copies=%d want 1", dm.DmCopies)
	}
}
