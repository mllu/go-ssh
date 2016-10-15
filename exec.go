package gossh

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"golang.org/x/crypto/ssh"
)

// Ref: https://github.com/hypersleep/easyssh/blob/master/easyssh.go

// Stream returns one channel that combines the stdout and stderr of the command
// as it is run on the remote machine, and another that sends true when the
// command is done. The sessions and channels will then be closed.
func Stream(session *ssh.Session, command string) (output chan string, done chan bool, err error) {
	// connect to both outputs (they are of type io.Reader)
	outReader, err := session.StdoutPipe()
	if err != nil {
		return output, done, err
	}
	errReader, err := session.StderrPipe()
	if err != nil {
		return output, done, err
	}
	// combine outputs, create a line-by-line scanner
	outputReader := io.MultiReader(outReader, errReader)
	err = session.Start(command)
	scanner := bufio.NewScanner(outputReader)
	// continuously send the command's output over the channel
	outputChan := make(chan string)
	done = make(chan bool)
	go func(scanner *bufio.Scanner, out chan string, done chan bool) {
		defer close(outputChan)
		defer close(done)
		for scanner.Scan() {
			outputChan <- scanner.Text()
		}
		// close all of our open resources
		done <- true
		session.Close()
	}(scanner, outputChan, done)
	return outputChan, done, err
}

// Runs command on remote machine and returns its stdout as a string
func Run(sess *ssh.Session, command string) (outStr string, err error) {
	outChan, doneChan, err := Stream(sess, command)
	if err != nil {
		return outStr, err
	}
	// read from the output channel until the done signal is passed
	stillGoing := true
	for stillGoing {
		select {
		case <-doneChan:
			stillGoing = false
		case line := <-outChan:
			outStr += line + "\n"
		}
	}
	// return the concatenation of all signals from the output channel
	return outStr, err
}

// Scp uploads sourceFile to remote machine like native scp console app.
func Scp(session *ssh.Session, sourceFile string) error {
	targetFile := filepath.Base(sourceFile)
	src, srcErr := os.Open(sourceFile)
	if srcErr != nil {
		return srcErr
	}

	srcStat, statErr := src.Stat()
	if statErr != nil {
		return statErr
	}

	go func() {
		w, _ := session.StdinPipe()
		fmt.Fprintln(w, "C0644", srcStat.Size(), targetFile)
		if srcStat.Size() > 0 {
			io.Copy(w, src)
			fmt.Fprint(w, "\x00")
			w.Close()
		} else {
			fmt.Fprint(w, "\x00")
			w.Close()
		}
	}()

	if err := session.Run(fmt.Sprintf("scp -t %s", targetFile)); err != nil {
		return err
	}
	return nil
}
