package shell

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"reflect"
	"strings"
	"syscall"

	"github.com/gruntwork-io/terragrunt/errors"
	"github.com/gruntwork-io/terragrunt/options"
)

// Run the given Terraform command
func RunTerraformCommand(terragruntOptions *options.TerragruntOptions, args ...string) error {
	return RunShellCommand(terragruntOptions, terragruntOptions.TerraformPath, args...)
}

// Run the given Terraform command and return the stdout as a string
func RunTerraformCommandAndCaptureOutput(terragruntOptions *options.TerragruntOptions, args ...string) (string, error) {
	return RunShellCommandAndCaptureOutput(terragruntOptions, terragruntOptions.TerraformPath, args...)
}

// Run the specified shell command with the specified arguments. Connect the command's stdin, stdout, and stderr to
// the currently running app.
func RunShellCommand(terragruntOptions *options.TerragruntOptions, command string, args ...string) error {
	terragruntOptions.Logger.Printf("Running Shell command: %s %s", command, strings.Join(args, " "))

	cmd := exec.Command(command, args...)

	cmd.Stdin = os.Stdin
	cmd.Env = toEnvVarsList(terragruntOptions.Env)

	// Terragrunt can run some commands (such as terraform remote config) before running the actual terraform
	// command requested by the user. The output of these other commands should not end up on stdout as this
	// breaks scripts relying on terraform's output.
	if !reflect.DeepEqual(terragruntOptions.TerraformCliArgs, args) {
		cmd.Stdout = cmd.Stderr
	}

	cmd.Dir = terragruntOptions.WorkingDir

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Println("**** got an error from opening StdoutPipe")
		return errors.WithStackTrace(err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		fmt.Println("**** got an error from opening StderrPipe")
		return errors.WithStackTrace(err)
	}


	if err := cmd.Start(); err != nil {
		// bad path, binary not executable, &c
		return errors.WithStackTrace(err)
	}
	cmdChannel := make(chan error)
	signalChannel := NewSignalsForwarder(forwardSignals, cmd, terragruntOptions.Logger, cmdChannel)
	defer signalChannel.Close()


	_, err = readStdoutAndStderr(terragruntOptions, stdout, stderr)
	if err != nil {
		fmt.Println("****** got an error from readStdoutAndStderr")
		return errors.WithStackTrace(err)
	}


	err = cmd.Wait()
	cmdChannel <- err

	return errors.WithStackTrace(err)
}

func toEnvVarsList(envVarsAsMap map[string]string) []string {
	envVarsAsList := []string{}
	for key, value := range envVarsAsMap {
		envVarsAsList = append(envVarsAsList, fmt.Sprintf("%s=%s", key, value))
	}
	return envVarsAsList
}

// Run the specified shell command with the specified arguments. Capture the command's stdout and return it as a
// string.
func RunShellCommandAndCaptureOutput(terragruntOptions *options.TerragruntOptions, command string, args ...string) (string, error) {
	stdout := new(bytes.Buffer)
	terragruntOptions.Logger.Printf("Running command: %s %s\n", command, strings.Join(args, " "))

	
	terragruntOptionsCopy := terragruntOptions.Clone(terragruntOptions.TerragruntConfigPath)
	terragruntOptionsCopy.Writer = stdout
	terragruntOptionsCopy.ErrWriter = stdout
	terragruntOptionsCopy.WorkingDir = terragruntOptions.WorkingDir



	err := RunShellCommand(terragruntOptionsCopy, command, args...)
	return stdout.String(), err
}

// Return the exit code of a command. If the error does not implement errors.IErrorCode or is not an exec.ExitError
// or errors.MultiError type, the error is returned.
func GetExitCode(err error) (int, error) {
	if exiterr, ok := errors.Unwrap(err).(errors.IErrorCode); ok {
		return exiterr.ExitStatus()
	}

	if exiterr, ok := errors.Unwrap(err).(*exec.ExitError); ok {
		status := exiterr.Sys().(syscall.WaitStatus)
		return status.ExitStatus(), nil
	}

	if exiterr, ok := errors.Unwrap(err).(errors.MultiError); ok {
		for _, err := range exiterr.Errors {
			exitCode, exitCodeErr := GetExitCode(err)
			if exitCodeErr == nil {
				return exitCode, nil
			}
		}
	}

	return 0, err
}

// This function captures stdout and stderr while still printing it to the stdout and stderr of this Go program
func readStdoutAndStderr(terragruntOptions *options.TerragruntOptions, stdout io.ReadCloser, stderr io.ReadCloser) (string, error) {
	allOutput := []string{}

	stdoutScanner := bufio.NewScanner(stdout)
	stderrScanner := bufio.NewScanner(stderr)

	for {
		if stdoutScanner.Scan() {
			text := stdoutScanner.Text()
			terragruntOptions.Logger.Println(text)
			allOutput = append(allOutput, text)
		} else if stderrScanner.Scan() {
			text := stderrScanner.Text()
			terragruntOptions.Logger.Println(text)
			allOutput = append(allOutput, text)
		} else {
			break
		}
	}

	if err := stdoutScanner.Err(); err != nil {
		return "", err
	}

	if err := stderrScanner.Err(); err != nil {
		return "", err
	}

	return strings.Join(allOutput, "\n"), nil
}


type SignalsForwarder chan os.Signal

// Forwards signals to a command, waiting for the command to finish.
func NewSignalsForwarder(signals []os.Signal, c *exec.Cmd, logger *log.Logger, cmdChannel chan error) SignalsForwarder {
	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, signals...)

	go func() {
		for {
			select {
			case s := <-signalChannel:
				logger.Printf("Forward signal %v to terraform.", s)
				err := c.Process.Signal(s)
				if err != nil {
					logger.Printf("Error forwarding signal: %v", err)
				}
			case <-cmdChannel:
				return
			}
		}
	}()

	return signalChannel
}

func (signalChannel *SignalsForwarder) Close() error {
	signal.Stop(*signalChannel)
	*signalChannel <- nil
	close(*signalChannel)
	return nil
}
