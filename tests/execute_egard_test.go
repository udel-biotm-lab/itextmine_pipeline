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
		Mongo_host:                  "127.0.0.1",
		Mongo_port:                  "27017",
		Pubtator_db:                 "pubtator",
		Pubtator_medline_collection: "medline.aligned",
		Medline_db:                  "medline_current",
		Medline_text_collection:     "text",
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
