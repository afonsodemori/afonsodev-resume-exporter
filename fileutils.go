package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
)

func AreFilesEqual(path1, path2 string) (bool, error) {
	file1, err := os.Open(path1)
	if err != nil {
		return false, fmt.Errorf("Failed to open file %s: %w", path1, err)
	}
	defer file1.Close()

	file2, err := os.Open(path2)
	if err != nil {
		return false, fmt.Errorf("Failed to open file %s: %w", path2, err)
	}
	defer file2.Close()

	const bufferSize = 4096
	buf1 := make([]byte, bufferSize)
	buf2 := make([]byte, bufferSize)

	for {
		n1, err1 := file1.Read(buf1)
		n2, err2 := file2.Read(buf2)

		if err1 != nil && err1 != io.EOF {
			return false, fmt.Errorf("Error reading file %s: %w", path1, err1)
		}
		if err2 != nil && err2 != io.EOF {
			return false, fmt.Errorf("Error reading file %s: %w", path2, err2)
		}

		if n1 != n2 || !bytes.Equal(buf1[:n1], buf2[:n2]) {
			return false, nil
		}

		if err1 == io.EOF && err2 == io.EOF {
			return true, nil
		}
	}
}

func DeleteFile(path string) error {
	err := os.Remove(path)
	if err != nil {
		return fmt.Errorf("Failed to delete file %s: %w", path, err)
	}
	return nil
}
