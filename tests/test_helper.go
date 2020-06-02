package tests

import (
	"bufio"
	"os"
)

func CountLines(filePath string) (int, error) {
	// open the input doc
	inputfile, inputFileOpenError := os.Open(filePath)
	if inputFileOpenError != nil {
		return 0, inputFileOpenError
	}
	defer inputfile.Close()

	// Start reading from the file with a reader.
	scanner := bufio.NewScanner(inputfile)
	const maxCapacity = 512 * 1024 // 512KB
	buffer := make([]byte, maxCapacity)
	scanner.Buffer(buffer, maxCapacity)

	// loop over the input
	lineIndex := 0
	for scanner.Scan() {
		lineIndex = lineIndex + 1
	}

	scanError := scanner.Err()
	if scanError != nil {
		return 0, scanError
	}

	return lineIndex, nil

}
