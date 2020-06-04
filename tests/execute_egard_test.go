package tests

import (
	"itextmine/misc"
	"itextmine/tools"
	"testing"

	"github.com/stretchr/testify/require"
)

// Test splitting of input doc
func TestExcuteEgard(t *testing.T) {
	inputDoc := "../data/egard/doc_in.json"
	workDir := "test_workdir"
	//outPutDir := "output_dir"
	toolName := "egard"
	//collectionType := "pmc"

	numOfParallelTasks := 3

	defer misc.CleanDir(workDir)

	// split the document
	splitErr := misc.SplitInputDoc(inputDoc, workDir, toolName, 20)
	require.Equal(t, nil, splitErr, splitErr)

	// Execute rlimsp
	eGardError := tools.ExecuteEGard(workDir, numOfParallelTasks)
	require.Equal(t, nil, eGardError, eGardError)

	// Reduce
	//reduceError := tools.Reduce(workDir, outPutDir, toolName, collectionType)
	//require.Equal(t, nil, reduceError, reduceError)

}
