package model

type LabelType int

const (
	LabelTypeGeneric     LabelType = 99
	LabelTypeComposition LabelType = 0
	LabelTypeTag         LabelType = 1
	LabelTypeJewelry     LabelType = 2
	LabelTypeInvitation  LabelType = 4
	LabelTypeShoe        LabelType = 5
	LabelTypeCDBox       LabelType = 11
)

type LayoutField struct {
	Name   string
	Size   int
	Offset int
}

type LayoutDefinition struct {
	TypeCode        string
	Name            string
	NumCol          int
	NumRow          int
	CopiesPerColumn int
	WidthMM         float64
	HeightMM        float64
	MarginLeft      float64
	MarginTop       float64
	SpacingCol      float64
	SpacingRow      float64
	Flip            int
	Rotation        int
	Landscape       int
	Fields          map[string]string
}

type LabelRecord struct {
	Fields map[string]string
}

type Label struct {
	Name         string
	Type         string
	PrinterName  string
	LayoutType   string
	TemplateName string
	Data         LabelRecord
	ImagePath    string
	Texts        []TextElement
	WMFSymbols   []WMFSymbol
	Objects      []PrintObject
}

type PrintObject struct {
	Type  string
	Text  TextElement
	WMF   WMFSymbol
	Shape ShapeElement
}

type ShapeElement struct {
	XMM      float64
	YMM      float64
	WidthMM  float64
	HeightMM float64
}

type TextElement struct {
	FileOffset int
	FEFlags    uint32
	FETag      uint32
	Text       string
	PayloadRaw []byte
	RTFRaw     []byte
	StyleByte  byte
	NextX      uint32
	NextY      uint32
	XMM        float64
	YMM        float64
	WidthMM    float64
	HeightMM   float64
	FontName   string
	FontSize   float64
	Bold       bool
	Italic     bool
	Underline  bool
	Align      string
}

type WMFSymbol struct {
	FileOffset int
	FilePath   string
	Embedded   []byte
	PreBlock   []byte
	StyleByte  byte
	NextX      uint32
	NextY      uint32
	XMM        float64
	YMM        float64
	WidthMM    float64
	HeightMM   float64
}

type PrintPage struct {
	LabelsPerPage int
	Overrides     map[int]int
}
