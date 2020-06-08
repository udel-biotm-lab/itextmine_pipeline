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
	egardDBParams := tools.EgardDBParams{
		MongoHost:                 "127.0.0.1",
		MongoPort:                 "27017",
		PubtatorDB:                "pubtator",
		PubtatorMedlineCollection: "medline.aligned",
		MedlineDB:                 "medline_current",
		MedlineTextCollection:     "text",
	}

	numberOfLines := 20
	numOfParallelTasks := 6

	//defer misc.CleanDir(workDir)

	// split the document
	splitErr := misc.SplitInputDoc(inputDoc, workDir, toolName, numberOfLines)
	require.Equal(t, nil, splitErr, splitErr)

	// Execute rlimsp
	eGardError := tools.ExecuteEGard(workDir, numOfParallelTasks, egardDBParams)
	require.Equal(t, nil, eGardError, eGardError)

	// Reduce
	//reduceError := tools.Reduce(workDir, outPutDir, toolName, collectionType)
	//require.Equal(t, nil, reduceError, reduceError)

}
