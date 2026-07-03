package core

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"
)

func HashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func ReadFontName(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	var head [12]byte
	io.ReadFull(f, head[:])
	tag := string(head[:4])
	switch tag {
	case "\x00\x01\x00\x00", "true", "OTTO":
		numTables := int(binary.BigEndian.Uint16(head[4:8]))
		for i := 0; i < numTables; i++ {
			var rec [16]byte
			io.ReadFull(f, rec[:])
			if string(rec[:4]) == "name" {
				offset := int64(binary.BigEndian.Uint32(rec[8:12]))
				length := int64(binary.BigEndian.Uint32(rec[12:16]))
				return readNameRecord(path, offset, length)
			}
		}
	}
	return "", fmt.Errorf("name table not found")
}

func readNameRecord(path string, offset, length int64) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	f.Seek(offset, 0)
	var format, count uint16
	binary.Read(f, binary.BigEndian, &format)
	binary.Read(f, binary.BigEndian, &count)
	var stringOffset uint16
	binary.Read(f, binary.BigEndian, &stringOffset)
	for i := uint16(0); i < count; i++ {
		var platform, encoding, language, nameID, nameLen, strOff uint16
		binary.Read(f, binary.BigEndian, &platform)
		binary.Read(f, binary.BigEndian, &encoding)
		binary.Read(f, binary.BigEndian, &language)
		binary.Read(f, binary.BigEndian, &nameID)
		binary.Read(f, binary.BigEndian, &nameLen)
		binary.Read(f, binary.BigEndian, &strOff)
		if nameID == 1 && platform == 3 && encoding == 1 && language == 1033 {
			cur, _ := f.Seek(0, io.SeekCurrent)
			f.Seek(offset+int64(stringOffset)+int64(strOff), 0)
			data := make([]byte, nameLen)
			io.ReadFull(f, data)
			f.Seek(cur, 0)
			runes := make([]rune, nameLen/2)
			for j := 0; j < len(data); j += 2 {
				runes[j/2] = rune(binary.BigEndian.Uint16(data[j:]))
			}
			return strings.TrimSpace(string(runes)), nil
		}
	}
	return "", fmt.Errorf("no suitable name")
}
