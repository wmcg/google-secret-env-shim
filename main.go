package main

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"os"
	"syscall"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"github.com/alecthomas/kong"
)

func accessSecretVersion(name string) string {
	ctx := context.Background()
	client, err := secretmanager.NewClient(ctx)
	if err != nil {
		fmt.Printf("## ERROR: failed to create secretmanager client: %v", err)
		os.Exit(1)
	}
	defer client.Close()

	req := &secretmanagerpb.AccessSecretVersionRequest{
		Name: name,
	}

	result, err := client.AccessSecretVersion(ctx, req)
	if err != nil {
		fmt.Printf("## ERROR: failed to access secret version: %v", err)
		os.Exit(1)
	}

	crc32c := crc32.MakeTable(crc32.Castagnoli)
	checksum := int64(crc32.Checksum(result.Payload.Data, crc32c))
	if checksum != *result.Payload.DataCrc32C {
		fmt.Printf("## ERROR: data corruption detected")
		os.Exit(1)
	}

	return string(result.Payload.Data[:])
}

func buildSecretPath(p string, sName string, sValue string) string {
	return fmt.Sprintf("projects/%s/secrets/%s/versions/%s", p, sName, sValue)
}

func strToEnvs(sData string) []string {
	var data map[string]string
	err := json.Unmarshal([]byte(sData), &data)
	if err != nil {
		fmt.Printf("## ERROR: couldnt unmarshal string '%v'", err)
		os.Exit(1)
	}

	var envs []string
	for key, value := range data {
		envs = append(envs, fmt.Sprintf("%s=%s", key, value))
	}

	return envs
}

func execProcess(execPath string, args []string, env []string) {
	fmt.Printf("## Executing: '%s %s'\n", execPath, args)

	err := syscall.Exec(execPath, args, env)
	if err != nil {
		fmt.Printf("## Error executing new process: %v\n", err)
		os.Exit(1)
	}
}

type CLI struct {
	Verbose       bool     `flag:"" short:"v" help:"When passed will print the secrets so use with caution"`
	SecretName    string   `flag:"" short:"n" required:"" env:"SECRET_NAME" help:"Secret Name. Required."`
	Project       string   `flag:"" short:"p" required:"" env:"PROJECT" help:"Project. Required."`
	SecretVersion string   `flag:"" name:"secret-version" env:"SECRET_VERSION" default:"latest" optional:"" help:"Secret version. Defaults to 'latest'"`
	Cmd           []string `arg:"" name:"cmd" required:"" help:"Path to binary and any options"`
}

func main() {
	var cli CLI
	kong.Parse(&cli)

	sValue := accessSecretVersion(fmt.Sprintf("projects/%s/secrets/%s/versions/%s", cli.Project, cli.SecretName, cli.SecretVersion))
	envs := strToEnvs(sValue)

	if cli.Verbose {
		fmt.Printf("## Found %d ENV VARs in the secret:\n", len(envs))
		for _, env := range envs {
			fmt.Printf("## %s\n", env)
		}
	}

	execProcess(cli.Cmd[0], cli.Cmd[1:], envs)
}
