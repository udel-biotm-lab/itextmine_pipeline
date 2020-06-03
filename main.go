package main

import (
	"errors"
	"itextmine/misc"
	"itextmine/tools"

	"github.com/jessevdk/go-flags"
)

type Options struct {
	Tool           string `short:"t" long:"toolname" description:"Name of the text mining tool to run. Options are rlimsp" required:"true"`
	Workdir        string `short:"w" long:"workdir" description:"Full path to the workdir. Please ensure that the user has rw access to the directory" required:"true"`
	InputDoc       string `short:"i" long:"inputfile" description:"Full path to the input file. Please ensure that the user has read access to the file" required:"true"`
	OutputDir      string `short:"o" long:"outputdir" description:"Full path to the output directory. Please ensure that the user has rw access to the directory" required:"true"`
	CollectionType string `short:"c" long:"collection" description:"Type of collection" required:"true"`
	NumberOfTask   int    `short:"n" long:"numtasks" description:"Number of parallel tasks" default:"10"`
	LinesPerTask   int    `short:"l" long:"linespertask" description:"Number of lines per tasks" default:"100"`
}

func main() {
	opts := Options{}

	// parse arguments
	_, err := flags.Parse(&opts)
	if err != nil {
		panic(err)
	}

	// split the input doc
	splitError := misc.SplitInputDoc(opts.InputDoc, opts.Workdir, opts.Tool, opts.LinesPerTask)
	if splitError != nil {
		panic(splitError)
	}

	// run tool based on arguments
	if opts.Tool == "rlimsp" {
		rlimspError := tools.ExecuteRlimsp(opts.Workdir, opts.NumberOfTask)
		if rlimspError != nil {
			panic(rlimspError)
		}
	}

	// reduce
	reduceError := tools.Reduce(opts.Workdir, opts.OutputDir, opts.Tool, opts.CollectionType)
	if reduceError != nil {
		panic(reduceError)
	}
}

func validateArguments(opt Options) error {
	tools := []string{"rlimsp"}

	if misc.StringInSlice(opt.Tool, tools) == false {
		// check tool names
		return errors.New(opt.Tool + " is not a valid toolname")
	} else if len(opt.InputDoc) == 0 {
		// check input path
		return errors.New("Input path cannot be empty")
	} else if len(opt.OutputDir) == 0 {
		// check output path
		return errors.New("Outdir path cannot be empty")
	} else if len(opt.OutputDir) == 0 {
		// check workdir
		return errors.New("Workdir path cannot be empty")
	} else {
		return nil
	}

}
