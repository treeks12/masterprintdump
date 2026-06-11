package etq

import (
	"errors"
	"fmt"
	"os"
)

var ErrUnknownObjects = errors.New("etq: unknown objects prevent write")

func CanWrite(doc *ETQFile) bool {
	return doc != nil && len(doc.UnknownObjects) == 0
}

func CopyETQ(srcPath, dstPath string) error {
	data, err := os.ReadFile(srcPath)
	if err != nil {
		return fmt.Errorf("copy ETQ: read source: %w", err)
	}
	if err := os.WriteFile(dstPath, data, 0644); err != nil {
		return fmt.Errorf("copy ETQ: write destination: %w", err)
	}
	return nil
}

func SaveETQ(doc *ETQFile, dstPath string) error {
	if doc == nil {
		return fmt.Errorf("save ETQ: nil document")
	}
	if len(doc.UnknownObjects) > 0 {
		return ErrUnknownObjects
	}
	if doc.FilePath == "" {
		return fmt.Errorf("save ETQ: missing source path")
	}
	return CopyETQ(doc.FilePath, dstPath)
}
