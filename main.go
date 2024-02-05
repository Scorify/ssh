package ssh

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"golang.org/x/crypto/ssh"
)

type Schema struct {
	Target         string `json:"target"`
	Port           int    `json:"port"`
	Username       string `json:"username"`
	Password       string `json:"password"`
	Command        string `json:"command"`
	ExpectedOutput string `json:"expected_output"`
}

func Run(ctx context.Context, config string) error {
	// Define a new Schema
	schema := Schema{}

	// Unmarshal the config to the Schema
	err := json.Unmarshal([]byte(config), &schema)
	if err != nil {
		return err
	}

	ssh_config := &ssh.ClientConfig{
		User: schema.Username,
		Auth: []ssh.AuthMethod{
			ssh.Password(schema.Password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	target := fmt.Sprintf("%s:%d", schema.Target, schema.Port)
	errChan := make(chan error)

	go func() {
		defer close(errChan)
		client, err := ssh.Dial("tcp", target, ssh_config)
		if err != nil {
			errChan <- err
			return
		}
		defer client.Close()

		session, err := client.NewSession()
		if err != nil {
			errChan <- err
			return
		}
		defer session.Close()

		output, err := session.CombinedOutput(schema.Command)
		if err != nil {
			errChan <- err
			return
		}

		outputString := strings.TrimSpace(string(output))
		expectedOutputString := strings.TrimSpace(schema.ExpectedOutput)

		if outputString != expectedOutputString {
			errChan <- fmt.Errorf("expected output \"%s\" but got \"%s\"", expectedOutputString, outputString)
			return
		}

		errChan <- nil
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errChan:
		return err
	}
}
