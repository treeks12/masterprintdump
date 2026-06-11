package render

const mmPerInch = 25.4

const (
	DeviceCapHorzSize = 4
	DeviceCapVertSize = 6
	DeviceCapHorzRes  = 8
	DeviceCapVertRes  = 10
)

type CadMapaDeviceCaps struct {
	HorzRes  int
	VertRes  int
	HorzSize int
	VertSize int
	DPIX     int
	DPIY     int
}

func CadMapaPxPerMm(resPx, sizeMm int) float64 {
	if sizeMm <= 0 {
		return 0
	}
	return float64(resPx) / float64(sizeMm)
}

func CadMapaPageSizeMM(c CadMapaDeviceCaps) (float64, float64) {
	if c.HorzSize > 0 && c.VertSize > 0 {
		return float64(c.HorzSize), float64(c.VertSize)
	}
	if c.DPIX > 0 && c.DPIY > 0 && c.HorzRes > 0 && c.VertRes > 0 {
		return float64(c.HorzRes) * mmPerInch / float64(c.DPIX), float64(c.VertRes) * mmPerInch / float64(c.DPIY)
	}
	return 0, 0
}

func PageSizeFromPhysicalPx(physW, physH, dpiX, dpiY int) (float64, float64) {
	if dpiX <= 0 || dpiY <= 0 {
		return 0, 0
	}
	return float64(physW) * mmPerInch / float64(dpiX), float64(physH) * mmPerInch / float64(dpiY)
}
