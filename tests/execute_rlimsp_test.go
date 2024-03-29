package tests

import (
	"itextmine/misc"
	"itextmine/tools"
	"testing"

	"github.com/stretchr/testify/require"
)

// Test splitting of input doc
func TestExcuteRlimsp(t *testing.T) {
	inputDoc := "../data/rlimsp/test_execute_doc_in.json"
	workDir := "test_workdir"
	outPutDir := "output_dir"

	numOfParallelTasks := 3

	defer misc.CleanDir(workDir)

	// split the document
	splitErr := misc.SplitInputDoc(inputDoc, workDir, "rlimsp", 20)
	require.Equal(t, nil, splitErr, splitErr)

	// Execute rlimsp
	rlimspError := tools.ExecuteRlimsp(workDir, numOfParallelTasks)
	require.Equal(t, nil, rlimspError, rlimspError)

	// Reduce
	reduceError := tools.Reduce(workDir, outPutDir, "rlimsp", "medline")
	require.Equal(t, nil, reduceError, reduceError)

}
