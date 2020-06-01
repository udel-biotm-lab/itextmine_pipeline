package tests

import (
	"itextmine/misc"
	"itextmine/rlimsp"
	"testing"

	"github.com/stretchr/testify/require"
)

// Test splitting of input doc
func TestExcuteRlimsp(t *testing.T) {
	inputDoc := "../data/rlimsp/test_execute_doc_in.json"
	workDir := "test_workdir"
	numOfParallelTasks := 3

	defer misc.CleanDir(workDir)

	// split the document
	splitErr := misc.SplitInputDoc(inputDoc, workDir, "rlimsp", 50)
	require.Equal(t, nil, splitErr, splitErr)

	// Execute rlimsp
	rlimspError := rlimsp.Execute(workDir, numOfParallelTasks)
	require.Equal(t, nil, rlimspError, rlimspError)

}
