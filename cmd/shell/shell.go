package main

import (
	"log"
	"os"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"

	gs "github.com/mllu/go-ssh"
)

func main() {
	sc := gs.NewConfig()
	sc.ParseCommandLine()

	sess, err := sc.NewSSHClientSession()
	if err != nil {
		log.Fatal(err)
	}
	defer sess.Close()

	// Set IO
	sess.Stdin = os.Stdin
	sess.Stdout = os.Stdout
	sess.Stderr = os.Stderr

	modes := ssh.TerminalModes{
		ssh.ECHO:          1,      // please print what I type
		ssh.ECHOCTL:       0,      // please don't print control chars
		ssh.TTY_OP_ISPEED: 115200, // baud in
		ssh.TTY_OP_OSPEED: 115200, // baud out
	}

	termFD := int(os.Stdin.Fd())

	w, h, _ := terminal.GetSize(termFD)

	termState, _ := terminal.MakeRaw(termFD)
	defer terminal.Restore(termFD, termState)

	if err := sess.RequestPty("xterm-256color", h, w, modes); err != nil {
		log.Fatalf("request for pseudo terminal failed: %s", err)
	}
	// Start remote shell
	if err := sess.Shell(); err != nil {
		log.Fatalf("failed to start shell: %s", err)
	}
	sess.Wait()
}
