### itextmine pipeline

This repo holds the source code for itextmine data processing pipeline

## Requirements
```
1. OS - Ubuntu 18.04 LTS
2. Programming language - Golang
```

## Instructions
```
1. Install golang from - https://golang.org/doc/install

2. git clone - https://github.com/udel-biotm-lab/itextmine_pipeline.git

3. cd itextmine_pipeline

4. go run main.go --help

5. Run all tests go test -v itextmine/tests

6. Run individual test go test -v itextmine/tests -run TestExcuteRlimsp
```

## Best practices
If you are developing a tool to integrate into the pipeline, please take a look at the [Wiki](https://github.com/udel-biotm-lab/itextmine_pipeline/wiki/Best-practices-for-developing-and-dockerizing-tools) to ensure that you follow the best practices to streamline the integration of the tool.
