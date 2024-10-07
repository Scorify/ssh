package ssh

import (
	"context"
	"fmt"
	"strings"

	"github.com/scorify/schema"
	"golang.org/x/crypto/ssh"
)

type Schema struct {
	Server         string `key:"server"`
	Port           int    `key:"port" default:"22"`
	Username       string `key:"username"`
	Password       string `key:"password"`
	Command        string `key:"command"`
	ExpectedOutput string `key:"expected_output"`
}

func Validate(config string) error {
	conf := Schema{}

	err := schema.Unmarshal([]byte(config), &conf)
	if err != nil {
		return err
	}

	if conf.Server == "" {
		return fmt.Errorf("server is required; got %q", conf.Server)
	}

	if conf.Port <= 0 || conf.Port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535; got %d", conf.Port)
	}

	if conf.Username == "" {
		return fmt.Errorf("username is required; got %q", conf.Username)
	}

	if conf.Password == "" {
		return fmt.Errorf("password is required; got %q", conf.Password)
	}

	if conf.Command == "" {
		return fmt.Errorf("command is required; got %q", conf.Command)
	}

	return nil
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
