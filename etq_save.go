//go:build windows

package main

import (
	"encoding/base64"
	"errors"
	"fmt"
	"math"
	"strings"

	"masterprint-native/internal/etq"
)

var (
	errETQSaveNoSource       = errors.New("etq save: no source path")
	errETQSaveUnknownObjects = errors.New("etq save: unknown objects")
	errETQSaveStructural     = errors.New("etq save: structural change")
	errETQSaveUnsupported    = errors.New("etq save: unsupported field change")
	errETQSaveVariableText   = errors.New("etq save: variable-length text")
)

const etqMMEpsilon = 0.005

type etqElementSnapshot struct {
	FileOffset int
	Type       string
	PayloadRaw string
	XMM        float64
	YMM        float64
	WidthMM    float64
	HeightMM   float64
	Text       string
	FontName   string
	FontSize   float64
	Bold       bool
	Italic     bool
	Underline  bool
	Align      string
	ImagePath  string
	SymbolName string
}

func snapshotETQElements(elements []LabelElement) []etqElementSnapshot {
	out := make([]etqElementSnapshot, 0, len(elements))
	for _, el := range elements {
		if el.Type == "preview" {
			continue
		}
		out = append(out, etqElementSnapshot{FileOffset: el.FileOffset, Type: el.Type, PayloadRaw: el.PayloadRaw, XMM: el.XMM, YMM: el.YMM, WidthMM: el.WidthMM, HeightMM: el.HeightMM, Text: el.Text, FontName: el.FontName, FontSize: el.FontSize, Bold: el.Bold, Italic: el.Italic, Underline: el.Underline, Align: el.Align, ImagePath: el.ImagePath, SymbolName: el.SymbolName})
	}
	return out
}

func (a *App) maybeSaveETQ(dstPath string) (bool, error) {
	if !envFlag("MASTERPRINT_ETQ_SAVE") {
		return false, nil
	}
	if err := a.saveETQPatched(dstPath); err != nil {
		return false, err
	}
	return true, nil
}

func (a *App) saveETQPatched(dstPath string) error {
	if a.etqSourcePath == "" {
		return fmt.Errorf("salvar ETQ: %w", errETQSaveNoSource)
	}
	if len(a.unknownObjects) > 0 {
		return fmt.Errorf("salvar ETQ: %w", errETQSaveUnknownObjects)
	}
	if err := preflightETQSave(a.etqBaseline, a.elements); err != nil {
		return fmt.Errorf("salvar ETQ: %w", err)
	}
	doc, err := etq.ParseETQ(a.etqSourcePath)
	if err != nil {
		return fmt.Errorf("salvar ETQ: reler origem: %w", err)
	}
	if err := etq.SavePatchedETQ(doc, dstPath, buildETQPatch(a.etqBaseline, a.elements)); err != nil {
		return fmt.Errorf("salvar ETQ: %w", err)
	}
	a.etqBaseline = snapshotETQElements(a.elements)
	if !strings.EqualFold(dstPath, a.etqSourcePath) {
		a.etqSourcePath = dstPath
	}
	return nil
}

func preflightETQSave(baseline []etqElementSnapshot, current []LabelElement) error {
	baseByOff := etqOffsetsFromSnapshot(baseline)
	curByOff, err := etqOffsetsFromElements(current)
	if err != nil {
		return err
	}
	if len(baseByOff) != len(curByOff) {
		return errETQSaveStructural
	}
	for off, base := range baseByOff {
		cur, ok := curByOff[off]
		if !ok || base.Type != cur.Type {
			return errETQSaveStructural
		}
		switch base.Type {
		case "text":
			if textFieldsChanged(base, cur) {
				return errETQSaveUnsupported
			}
			if base.Text != cur.Text {
				if err := assertSamePlainLatin1PayloadLen(base, cur.Text); err != nil {
					return err
				}
			}
		case "image":
			if base.ImagePath != cur.ImagePath || base.SymbolName != cur.SymbolName {
				return errETQSaveUnsupported
			}
		default:
			return errETQSaveStructural
		}
	}
	return nil
}

func buildETQPatch(baseline []etqElementSnapshot, current []LabelElement) func(*etq.Patcher) error {
	baseByOff := etqOffsetsFromSnapshot(baseline)
	return func(p *etq.Patcher) error {
		for off, base := range baseByOff {
			cur, ok := findElementByOffset(current, off)
			if !ok {
				return errETQSaveStructural
			}
			switch base.Type {
			case "text":
				if !nearETQMM(base.XMM, cur.XMM) || !nearETQMM(base.YMM, cur.YMM) {
					if err := p.PatchTextPosition(off, mmToETQRaw(cur.XMM), mmToETQRaw(cur.YMM)); err != nil {
						return err
					}
				}
				if base.Text != cur.Text {
					if err := p.PatchTextPayloadLatin1(off, cur.Text); err != nil {
						return err
					}
				}
			case "image":
				if !nearETQMM(base.XMM, cur.XMM) || !nearETQMM(base.YMM, cur.YMM) || !nearETQMM(base.WidthMM, cur.WidthMM) || !nearETQMM(base.HeightMM, cur.HeightMM) {
					if err := p.PatchWMFRect(off, mmToETQRaw(cur.WidthMM), mmToETQRaw(cur.HeightMM), mmToETQRaw(cur.XMM), mmToETQRaw(cur.YMM)); err != nil {
						return err
					}
				}
			}
		}
		return nil
	}
}

func etqOffsetsFromSnapshot(snaps []etqElementSnapshot) map[int]etqElementSnapshot {
	out := make(map[int]etqElementSnapshot, len(snaps))
	for _, s := range snaps {
		if s.FileOffset > 0 {
			out[s.FileOffset] = s
		}
	}
	return out
}

func etqOffsetsFromElements(elements []LabelElement) (map[int]LabelElement, error) {
	out := map[int]LabelElement{}
	for _, el := range elements {
		if el.Type == "preview" {
			continue
		}
		if el.FileOffset <= 0 {
			return nil, errETQSaveStructural
		}
		if _, exists := out[el.FileOffset]; exists {
			return nil, errETQSaveStructural
		}
		out[el.FileOffset] = el
	}
	return out, nil
}

func findElementByOffset(elements []LabelElement, offset int) (LabelElement, bool) {
	for _, el := range elements {
		if el.FileOffset == offset {
			return el, true
		}
	}
	return LabelElement{}, false
}

func textFieldsChanged(base etqElementSnapshot, cur LabelElement) bool {
	return base.FontName != cur.FontName || !nearETQMM(base.FontSize, cur.FontSize) || !nearETQMM(base.WidthMM, cur.WidthMM) || !nearETQMM(base.HeightMM, cur.HeightMM) || base.Bold != cur.Bold || base.Italic != cur.Italic || base.Underline != cur.Underline || base.Align != cur.Align
}

func assertSamePlainLatin1PayloadLen(base etqElementSnapshot, after string) error {
	basePayload, err := latin1Payload(base.Text)
	if err != nil {
		return err
	}
	rawLen := len(basePayload)
	if base.PayloadRaw != "" {
		raw, err := base64.StdEncoding.DecodeString(base.PayloadRaw)
		if err != nil {
			return errETQSaveUnsupported
		}
		rawLen = len(raw)
		if len(basePayload) != rawLen {
			return errETQSaveUnsupported
		}
	}
	afterPayload, err := latin1Payload(after)
	if err != nil {
		return err
	}
	if len(afterPayload) != rawLen {
		return errETQSaveVariableText
	}
	return nil
}

func latin1Payload(text string) ([]byte, error) {
	out := make([]byte, 0, len(text))
	for _, r := range text {
		if r > 0xff {
			return nil, errETQSaveUnsupported
		}
		out = append(out, byte(r))
	}
	return out, nil
}

func mmToETQRaw(mm float64) uint32 {
	if mm < 0 {
		return 0
	}
	return uint32(math.Round(mm * 100))
}

func nearETQMM(a, b float64) bool {
	return math.Abs(a-b) < etqMMEpsilon
}
