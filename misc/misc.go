package misc

import (
	"bufio"
	"fmt"
	"os"
	"path"
	"strings"
)

func StringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func SplitInputDoc(inputDocPath string, workdirPath string, toolName string, numberOfLines int) error {
	// generate the path for workdir
	toolWorkDirPath := path.Join(workdirPath, toolName)

	// check if exists
	toolWorkDirExist, err := pathExists(toolWorkDirPath)
	if err != nil {
		return err
	}

	// if does not exist create it
	if toolWorkDirExist == false {
		mkDirErr := os.MkdirAll(toolWorkDirPath, os.FileMode(0777))
		if mkDirErr != nil {
			panic(mkDirErr)
		}
	}

	// clean the work dir
	cleanError := cleanDir(toolWorkDirPath)
	if cleanError != nil {
		return cleanError
	}

	// open the input doc
	inputfile, inputFileOpenError := os.Open(inputDocPath)
	if inputFileOpenError != nil {
		return inputFileOpenError
	}
	defer inputfile.Close()

	// Start reading from the file with a reader.
	scanner := bufio.NewScanner(inputfile)
	const maxCapacity = 512 * 1024 // 512KB
	buffer := make([]byte, maxCapacity)
	scanner.Buffer(buffer, maxCapacity)

	// constraints
	lineIndex := 0
	taskIndex := 0
	linesBuffer := make([]string, 0)

	// loop over the input
	for scanner.Scan() {
		lineIndex = lineIndex + 1
		linesBuffer = append(linesBuffer, scanner.Text())

		if lineIndex%numberOfLines == 0 {
			// rest line index to 0
			lineIndex = 0

			// write to disk
			writeLines(taskIndex, linesBuffer, toolWorkDirPath)

			// increment the task index
			taskIndex = taskIndex + 1

			// reset the lines buffer
			linesBuffer = make([]string, 0)

		}
	}

	scanError := scanner.Err()
	if scanError != nil {
		return scanError
	}

	// if the lines buffer is not empty then write the remaining lines
	if len(linesBuffer) > 0 {
		writeLines(taskIndex, linesBuffer, toolWorkDirPath)
	}

	return nil
}

func pathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return true, err
}

func cleanDir(dirPath string) error {
	dir, dirErr := os.Open(dirPath)
	if dirErr != nil {
		return dirErr
	}

	subDirs, subDirErr := dir.Readdirnames(0)
	if subDirErr != nil {
		return subDirErr
	}

	for index := range subDirs {
		subDir := subDirs[index]
		subDirPath := path.Join(dirPath, subDir)
		subDirRemoveError := os.RemoveAll(subDirPath)
		if subDirRemoveError != nil {
			return subDirRemoveError
		}
	}

	return nil
}

func writeLines(taskIndex int, lines []string, toolWorkDirPath string) error {
	// create a folder for this task
	taskFolderName := path.Join(toolWorkDirPath, fmt.Sprintf("task_%d", taskIndex))
	taskFolderError := os.Mkdir(taskFolderName, os.FileMode(0777))
	if taskFolderError != nil {
		return taskFolderError
	}

	//  create a file for writing
	taskInputFilePath := path.Join(taskFolderName, fmt.Sprintf("input_%d.json", taskIndex))
	taskInputFile, err := os.Create(taskInputFilePath)
	if err != nil {
		return err
	}
	defer taskInputFile.Close()

	// join all the lines into one string
	joinedLines := strings.Join(lines, "\n")
	joinedLines = joinedLines + "\n"

	// create the witer
	taskInputwriter := bufio.NewWriter(taskInputFile)
	_, writeErr := taskInputwriter.WriteString(joinedLines)
	if writeErr != nil {
		return writeErr
	}

	return nil

}
