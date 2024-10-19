package utils

import (
	"encoding/binary"
	"fmt"
	"os"
)

func ReadBinary[T any](filename string) ([]T, error) {

	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %v", err)
	}

	numElements := int(fileInfo.Size()) / binary.Size(new(T))
	data := make([]T, numElements)

	err = binary.Read(file, binary.LittleEndian, &data)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %v", err)
	}

	return data, nil
}

func WriteBinary[T any](filename string, data []T) error {

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file: %v", err)
	}
	defer file.Close()

	err = binary.Write(file, binary.LittleEndian, data)
	if err != nil {
		return fmt.Errorf("failed to write file: %v", err)
	}

	return nil
}
