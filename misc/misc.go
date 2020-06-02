package misc

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/docker/docker/client"
)

func CreateDockerClient() *client.Client {
	cli, err := client.NewEnvClient()
	if err != nil {
		panic(err)
	}
	return cli
}

func StringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func CheckOutput(outputJson string) error {
	fileStat, err := os.Stat(outputJson)
	if err != nil {
		return err
	}

	if fileStat.Size() == 0 {
		return errors.New(fmt.Sprintf("%s is empty", outputJson))
	} else {
		return nil
	}

}

func Shellout(command string) (error, string, string) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd := exec.Command("bash", "-c", command)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return err, stdout.String(), stderr.String()
}
