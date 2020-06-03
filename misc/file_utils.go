package misc

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path"
	"strings"
)

func SplitInputDoc(inputDocPath string, workdirPath string, toolName string, numberOfLines int) error {
	// generate the path for workdir
	toolWorkDirPath := path.Join(workdirPath, toolName)

	// check if exists
	toolWorkDirExist, _ := PathExists(toolWorkDirPath)

	// if does not exist create it
	if toolWorkDirExist == false {
		mkDirErr := os.MkdirAll(toolWorkDirPath, os.FileMode(0777))
		if mkDirErr != nil {
			return mkDirErr
		}
	}

	// clean the work dir
	cleanError := CleanDir(toolWorkDirPath)
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

	log.Println(fmt.Sprintf("Splitting input file to - %s", toolWorkDirPath))

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

func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	} else {
		return false, err
	}
}

func GetSubDirNames(dirPath string) (*[]string, error) {
	dir, dirErr := os.Open(dirPath)
	if dirErr != nil {
		return nil, dirErr
	}

	subDirs, subDirErr := dir.Readdirnames(0)
	if subDirErr != nil {
		return nil, subDirErr
	}
	return &subDirs, nil
}

func CleanDir(dirPath string) error {

	subDirs, err := GetSubDirNames(dirPath)
	if err != nil {
		return err
	}

	for index := range *subDirs {
		subDir := (*subDirs)[index]
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
	taskInputFilePath := path.Join(taskFolderName, "input.json")
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

	// wirte string
	_, writeErr := taskInputwriter.WriteString(joinedLines)
	if writeErr != nil {
		return writeErr
	}

	// flush the writer
	flushError := taskInputwriter.Flush()
	if flushError != nil {
		return flushError
	}

	return nil

}

func TouchFile(filePath string) error {
	// create the output.json
	file, fileCreateError := os.Create(filePath)
	if fileCreateError != nil {
		return fileCreateError
	}
	file.Close()
	return nil
}

func CreateFolderIfNotExists(folderPath string) error {
	// check if already exists
	folderExists, _ := PathExists(folderPath)

	// if does not exist, create it
	if folderExists == false {
		mkDirErr := os.MkdirAll(folderPath, os.FileMode(0777))
		if mkDirErr != nil {
			return mkDirErr
		}
	}

	return nil
}
