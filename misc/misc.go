package misc

import (
	"errors"
	"fmt"
	"os"

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
