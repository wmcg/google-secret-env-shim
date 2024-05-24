package main

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"slices"
	"strconv"
	"syscall"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"google.golang.org/api/iterator"
)

// listSecrets lists all secrets in the given project.
func listSecrets(w io.Writer, project string, filter string) []string {
	// parent := "projects/my-project"

	// Create the client.
	ctx := context.Background()
	client, err := secretmanager.NewClient(ctx)
	if err != nil {
		panic(fmt.Errorf("failed to create secretmanager client: %w", err))
	}
	defer client.Close()

	// Build the request.
	req := &secretmanagerpb.ListSecretsRequest{
		Parent: project,
		Filter: filter,
	}

	// Call the API.
	it := client.ListSecrets(ctx, req)
	var secrets []string
	for {
		resp, err := it.Next()
		if err == iterator.Done {
			break
		}

		if err != nil {
			panic(fmt.Errorf("failed to list secrets: %w", err))
		}
		secrets = append(secrets, resp.Name)
		fmt.Fprintf(w, "Found secret %s\n", resp.Name)
	}

	return secrets
}

func listSecretVersions(w io.Writer, parent string) error {
	// parent := "projects/my-project/secrets/my-secret"

	// Create the client.
	ctx := context.Background()
	client, err := secretmanager.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to create secretmanager client: %w", err)
	}
	defer client.Close()

	// Build the request.
	req := &secretmanagerpb.ListSecretVersionsRequest{
		Parent: parent,
	}

	// Call the API.
	it := client.ListSecretVersions(ctx, req)
	for {
		resp, err := it.Next()
		if err == iterator.Done {
			break
		}

		if err != nil {
			return fmt.Errorf("failed to list secret versions: %w", err)
		}

		fmt.Fprintf(w, "Found secret version %s with state %s\n",
			resp.Name, resp.State)
	}

	return nil
}

func getSecretVersion(w io.Writer, name string) *secretmanagerpb.SecretVersion {
	// name := "projects/my-project/secrets/my-secret"

	// Create the client.
	ctx := context.Background()
	client, err := secretmanager.NewClient(ctx)
	if err != nil {
		panic(fmt.Errorf("failed to create secretmanager client: %w", err))
	}
	defer client.Close()

	// Build the request.
	req := &secretmanagerpb.GetSecretVersionRequest{
		Name: name,
	}

	// Call the API.
	result, err := client.GetSecretVersion(ctx, req)
	if err != nil {
		panic(fmt.Errorf("failed to get secret: %w", err))
	}

	// replication := result.Replication.Replication
	create_time := result.CreateTime
	fmt.Fprintf(w, "Found secret %s with created '%s' \n", result.Name, create_time)
	return result
}

func accessSecretVersion(name string) string {
	// name := "projects/my-project/secrets/my-secret/versions/5"
	// name := "projects/my-project/secrets/my-secret/versions/latest"

	// Create the client.
	ctx := context.Background()
	client, err := secretmanager.NewClient(ctx)
	if err != nil {
		panic(fmt.Errorf("failed to create secretmanager client: %w", err))
	}
	defer client.Close()

	// Build the request.
	req := &secretmanagerpb.AccessSecretVersionRequest{
		Name: name,
	}

	// Call the API.
	result, err := client.AccessSecretVersion(ctx, req)
	if err != nil {
		panic(fmt.Errorf("failed to access secret version: %w", err))
	}

	// Verify the data checksum.
	crc32c := crc32.MakeTable(crc32.Castagnoli)
	checksum := int64(crc32.Checksum(result.Payload.Data, crc32c))
	if checksum != *result.Payload.DataCrc32C {
		panic(fmt.Errorf("data corruption detected"))
	}

	return string(result.Payload.Data[:])
}

func buildSecretPath(p string, sn string, sv string) string {
	if sv == "" {
		return fmt.Sprintf("projects/%s/secrets/%s/versions/latest", p, sn)
	} else {
		v, err := strconv.Atoi(sv)
		if err != nil {
			panic(fmt.Errorf("unable to convert secret '%s' version to integer error '%w'", sv, err))
		}
		return fmt.Sprintf("projects/%s/secrets/%s/versions/%d", p, sn, v)
	}
}

func strToEnvs(sData string) []string {
	// Unmarshal the JSON data into a map[string]string
	var data map[string]string
	err := json.Unmarshal([]byte(sData), &data)
	if err != nil {
		panic(fmt.Errorf("couldnt unmarshal string '%w'", err))
	}

	// Create a slice to store the formatted strings
	var envs []string

	// Iterate over the map and format the key-value pairs
	for key, value := range data {
		envs = append(envs, fmt.Sprintf("%s=%s", key, value))
	}

	return envs
}

func execProcess(execPath string, args []string, env []string) {
	// Print a message before executing the new process
	fmt.Printf("## Executing: '%s %s'\n", execPath, args)
	finalArgs := slices.Insert(args, 0, "python3")
	// Execute the new process
	err := syscall.Exec(execPath, finalArgs, env)
	if err != nil {
		// If there is an error, print it and exit
		fmt.Printf("Error executing new process: %v\n", err)
		os.Exit(1)
	}
}

func main() {

	project := "skim-stage-k8-misc"

	f := os.Getenv("FILTER")
	if f == "" {
		fmt.Println("## FILTER not set")
	}

	an := os.Getenv("APP_NAME")
	if an == "" {
		panic(fmt.Errorf("APP_NAME not set"))
	}

	appPath := os.Getenv("APP_BINARY_PATH")
	if appPath == "" {
		panic(fmt.Errorf("APP_BINARY_PATH not set"))
	}

	sv := os.Getenv("SECRET_VERSION")
	spath := buildSecretPath(project, an, sv)

	fmt.Printf("## Getting secret: '%s'\n", spath)
	sValue := accessSecretVersion(spath)

	envs := strToEnvs(sValue)
	appArgs := []string{"/host_app/main.py"}
	execProcess(appPath, appArgs, envs)

}
