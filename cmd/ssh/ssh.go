package main

import (
	"flag"
	"fmt"
	"log"

	gs "github.com/mllu/go-ssh"
)

func main() {
	cmd := flag.String("cmd", "pwd", "command to execute on remote server")
	sc := gs.NewConfig()
	sc.ParseCommandLine()

	sess, err := sc.NewSSHClientSession()
	if err != nil {
		log.Fatal(err)
	}
	defer sess.Close()
	rsp, err := gs.Run(sess, *cmd)
	if err != nil {
		log.Printf("remote command %s failed with error %v\n", *cmd, err)
		return
	}
	fmt.Print(rsp)
}
