package main

import (
	"errors"
	"itextmine/misc"

	"github.com/jessevdk/go-flags"
)

type Options struct {
	Tool         string `short:"t" long:"toolname" description:"Name of the text mining tool to run. Options are rlimsp, efip" required:"true"`
	InputDoc     string `short:"i" long:"inputfile" description:"Full path to the input file" required:"true"`
	OutputDoc    string `short:"o" long:"outputfile" description:"Full path to the output file" required:"true"`
	Workdir      string `short:"w" long:"workdir" description:"Full path to the workdir.Please ensure that the user has rw access to the directory" required:"true"`
	NumberOfTask int    `short:"n" long:"numtasks" description:"Number of parallel tasks" default:"10"`
	LinesPerTask int    `short:"l" long:"linespertask" description:"Number of lines per tasks" default:"100"`
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
		//rlimsp.Execute(opts.InputDoc, opts.OutputDoc, opts.Workdir, opts.NumberOfTask)
	}

}

func validateArguments(opt Options) error {
	tools := []string{"rlimsp", "efip"}

	if misc.StringInSlice(opt.Tool, tools) == false {
		// check tool names
		return errors.New(opt.Tool + " is not a valid toolname")
	} else if len(opt.InputDoc) == 0 {
		// check input path
		return errors.New("Input path cannot be empty")
	} else if len(opt.OutputDoc) == 0 {
		// check output path
		return errors.New("Output path cannot be empty")
	} else if len(opt.OutputDoc) == 0 {
		// check workdir
		return errors.New("Workdir path cannot be empty")
	} else {
		return nil
	}

}
