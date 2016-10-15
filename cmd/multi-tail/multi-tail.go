package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os/user"
	"strings"

	"golang.org/x/crypto/ssh"

	gs "github.com/mllu/go-ssh"
)

func TailLog(name, logFile string, client *ssh.Client, lines chan<- string) {
	sess, _ := client.NewSession()
	defer sess.Close()

	out, _ := sess.StdoutPipe()

	scanner := bufio.NewScanner(out)
	scanner.Split(bufio.ScanLines)

	sess.Start("tail -f " + logFile)

	for scanner.Scan() {
		lines <- fmt.Sprintf("[%s] %s", name, scanner.Text())
	}

	sess.Wait()
}

func MultiTail(bastion *ssh.Client, remoteAddrs []string, remoteUser, logFile string) {
	usr, _ := user.Current()
	lines := make(chan string)
	for _, remote := range remoteAddrs {
		part := strings.Split(remote, ":")
		if len(part) != 2 {
			return
		}
		host, port := part[0], part[1]
		sc := &gs.SSHConfig{
			User:    remoteUser,
			Host:    host,
			Port:    port,
			KeyFile: usr.HomeDir + "/.ssh/id_rsa",
		}
		auths, err := sc.GetAuthMethods()
		if err != nil {
			log.Fatal(err)
		}
		cfg := &ssh.ClientConfig{
			User: sc.User,
			Auth: auths,
		}
		go TailLog(
			remote,
			logFile,
			gs.Proxy(bastion, remote, cfg),
			lines,
		)
	}

	for l := range lines {
		log.Print(l)
	}
}

type StringArray []string

func (a *StringArray) Set(s string) error {
	*a = append(*a, s)
	return nil
}

func (a *StringArray) String() string {
	return strings.Join(*a, ",")
}

func main() {
	usr, _ := user.Current()
	sc := gs.NewConfig()
	remoteHosts := StringArray{}
	logFile := flag.String("log_file", "", "full path for log file to tail")
	flag.Var(&remoteHosts, "remote_address",
		"remote address including port, eg. foo.com:22, may be given multiple times")
	remoteUser := flag.String("remote_user", usr.Username, "username to login to remote hosts")
	sc.ParseCommandLine()

	cli, err := sc.NewSSHClient()
	if err != nil {
		log.Fatal(err)
	}
	defer cli.Close()
	hosts := strings.Split(remoteHosts.String(), ",")
	MultiTail(cli, hosts, *remoteUser, *logFile)
}
