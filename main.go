package main

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"github.com/alecthomas/kong"
)

type CLI struct {
	Verbose       bool     `flag:"" short:"v" help:"When passed will print the secrets so use with caution"`
	SecretName    string   `flag:"" short:"n" optional:"" env:"SECRET_NAME" help:"Secret Name. Required when in 'api' SECRET_MODE"`
	Project       string   `flag:"" short:"p" optional:"" env:"PROJECT" help:"Project. Required when in 'api' SECRET_MODE"`
	SecretVersion string   `flag:"" name:"secret-version" env:"SECRET_VERSION" default:"latest" optional:"" help:"Secret version. Defaults to 'latest'"`
	SecretMode    string   `flag:"" name:"secret-mode" env:"SECRET_MODE" default:"file" optional:"" help:"Secret mode. Can be 'file' (default) or 'api'"`
	Cmd           []string `arg:"" name:"cmd" required:"" help:"Path to binary and any options"`
}

func main() {
	var cli CLI
	kong.Parse(&cli)
	cmdPath := cli.Cmd[0]
	cmdArgs := cli.Cmd[1:]

	secretsPath := "/var/secrets/..data/"

	// we want currently set env vars as well as any additional we will add
	currentEnv := os.Environ()

	var envs []string
	if cli.SecretMode == "file" {
		envs = readFromFiles(secretsPath)
	} else if cli.SecretMode == "api" {
		sValue := accessSecretVersion(fmt.Sprintf("projects/%s/secrets/%s/versions/%s", cli.Project, cli.SecretName, cli.SecretVersion))
		envs = strToEnvs(sValue)
		if cli.Verbose {
			fmt.Printf("## gses - Found %d ENV VARs in the secret:\n", len(envs))
			for _, env := range envs {
				fmt.Printf("## gses - %s\n", env)
			}
		}
	} else {
		fmt.Printf("## gses - Must pass either 'file' or 'api' to secret-mode")
	}
	execBinary(cmdPath, cmdArgs, append(envs, currentEnv...))

}

func accessSecretVersion(name string) string {
	ctx := context.Background()
	client, err := secretmanager.NewClient(ctx)
	if err != nil {
		fmt.Printf("## gses - ERROR: failed to create secretmanager client: %v", err)
		os.Exit(1)
	}
	defer client.Close()

	req := &secretmanagerpb.AccessSecretVersionRequest{
		Name: name,
	}

	result, err := client.AccessSecretVersion(ctx, req)
	if err != nil {
		fmt.Printf("## gses - ERROR: failed to access secret version: %v", err)
		os.Exit(1)
	}

	crc32c := crc32.MakeTable(crc32.Castagnoli)
	checksum := int64(crc32.Checksum(result.Payload.Data, crc32c))
	if checksum != *result.Payload.DataCrc32C {
		fmt.Printf("## gses - ERROR: data corruption detected")
		os.Exit(1)
	}

	return string(result.Payload.Data[:])
}

func strToEnvs(sData string) []string {
	var data map[string]string
	err := json.Unmarshal([]byte(sData), &data)
	if err != nil {
		fmt.Printf("## gses - ERROR: couldnt unmarshal string '%v'", err)
		os.Exit(1)
	}

	var envs []string
	for key, value := range data {
		envs = append(envs, fmt.Sprintf("%s=%s", key, value))
	}

	return envs
}

func execBinary(execPath string, args []string, env []string) {
	fmt.Printf("## Executing: '%s' with args '%s' and '%d' env vars\n", execPath, args, len(env))

	// when no args passed argv[0] must be set to binary name
	var argv []string
	if len(args) == 0 {
		argv = []string{execPath}
	} else {
		argv = args
	}
	err := syscall.Exec(execPath, argv, env)
	if err != nil {
		fmt.Printf("## Error executing new process: %v\n", err)
		os.Exit(1)
	}
}

func readFromFiles(secretsPath string) []string {
	var envs []string

	walkErr := filepath.WalkDir(secretsPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("## gses - error accessing path %s: %v", path, err)
		}

		if d.IsDir() {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("## gses - error opening file %s: %v", path, err)
		}
		defer file.Close()

		secret, err := io.ReadAll(file)
		if err != nil {
			return fmt.Errorf("## gses -  error reading file %s: %v", path, err)
		}

		envs = append(envs, fmt.Sprintf("%s=%s", filepath.Base(path), string(secret)))

		return nil
	})
	if walkErr != nil {
		fmt.Printf("## gses - Error walking through secretsPath: %v\n", walkErr)
	}

	return envs
}

func readCurrentEnvs() {
	envVars := os.Environ() // Returns a slice of strings in "KEY=VALUE" format
	for _, env := range envVars {
		fmt.Println(env)
	}
}
