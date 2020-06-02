package tests

import (
	"itextmine/misc"
	"path"
	"testing"

	"github.com/stretchr/testify/require"
)

// Test splitting of input doc
func TestInputDocSplit(t *testing.T) {
	inputDoc := "../data/rlimsp/test_split_doc_in.json"
	workDir := "test_workdir"
	defer misc.CleanDir(workDir)

	// split the document
	splitErr := misc.SplitInputDoc(inputDoc, workDir, "rlimsp", 100)
	require.Equal(t, nil, splitErr, splitErr)

	// verify if proper number of tasks folder were created
	taskDirNames, taskDirNamesErr := misc.GetSubDirNames(path.Join(workDir, "rlimsp"))
	require.Equal(t, nil, taskDirNamesErr, taskDirNamesErr)
	require.Equal(t, len(*taskDirNames), 11, "Improper number of task folders")

	// count the lines in last task folder
	lastInputJsonPath := path.Join(workDir, "rlimsp", "task_10", "input.json")
	lineCount, lineCountError := CountLines(lastInputJsonPath)
	require.Equal(t, nil, lineCountError, lineCountError)
	require.Equal(t, 20, lineCount)
}
