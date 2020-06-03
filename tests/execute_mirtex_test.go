package tests

import (
	"itextmine/misc"
	"itextmine/tools"
	"testing"

	"github.com/stretchr/testify/require"
)

// Test splitting of input doc
func TestExcuteMirtex(t *testing.T) {
	inputDoc := "../data/mirtex/test_doc_in_pmc.json"
	workDir := "test_workdir"
	outPutDir := "output_dir"
	toolName := "mirtex"
	collectionType := "pmc"

	numOfParallelTasks := 3

	defer misc.CleanDir(workDir)

	// split the document
	splitErr := misc.SplitInputDoc(inputDoc, workDir, toolName, 20)
	require.Equal(t, nil, splitErr, splitErr)

	// Execute rlimsp
	rlimspError := tools.ExecuteMirtex(workDir, numOfParallelTasks)
	require.Equal(t, nil, rlimspError, rlimspError)

	// Reduce
	reduceError := tools.Reduce(workDir, outPutDir, toolName, collectionType)
	require.Equal(t, nil, reduceError, reduceError)

}
