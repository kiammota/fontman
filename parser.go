package main

import (
	"fmt"
	"os"

	"golang.org/x/image/font/sfnt"
)

func GetFontDetails(path string) (family, subFamily, fullName, version string, err error) {
	f, err := os.Open(path)
	if err != nil {
		return "", "", "", "", err
	}
	defer f.Close()

	font, err := sfnt.ParseReaderAt(f)
	if err != nil {
		return "", "", "", "", fmt.Errorf("failed to parse font binary: %w", err)
	}

	var buf sfnt.Buffer

	extractName := func(id sfnt.NameID) string {
		str, err := font.Name(&buf, id)
		if err != nil {
			return "Unknown"
		}
		return str
	}

	family = extractName(sfnt.NameIDFamily)
	subFamily = extractName(sfnt.NameIDSubfamily)
	fullName = extractName(sfnt.NameIDFull)
	version = extractName(sfnt.NameIDVersion)

	return family, subFamily, fullName, version, nil
}
