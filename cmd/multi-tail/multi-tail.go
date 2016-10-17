package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"os/user"
	"strings"
	"syscall"

	"golang.org/x/crypto/ssh"

	gs "github.com/mllu/go-ssh"
)

func TailLog(name, logFile string, onlyNewLine bool, client *ssh.Client, lines chan<- string) {
	sess, _ := client.NewSession()
	defer sess.Close()

	out, err := sess.StdoutPipe()
	if err != nil {
		log.Printf("Unable to setup stdout for session: %v", err)
		return
	}

	scanner := bufio.NewScanner(out)
	scanner.Split(bufio.ScanLines)

	sess.Start("tail -f " + logFile)

	lineCnt := 1
	for scanner.Scan() {
		if onlyNewLine && lineCnt <= 10 {
			lineCnt++
			continue
		}
		lines <- fmt.Sprintf("[%s] %s", name, scanner.Text())
	}

	sess.Wait()
}

func MultiTail(bastion *ssh.Client, remoteAddrs []string, remoteUser, logFile string, onlyNewLine bool, done chan bool) {
	lines := make(chan string)
	defer close(lines)
	termChan := make(chan os.Signal, 1)
	signal.Notify(termChan, syscall.SIGINT, syscall.SIGTERM)

	cfgs, err := gs.GetMultiCliConfig(remoteAddrs, remoteUser)
	if err != nil {
		log.Println(err)
		return
	}

	clients := []*ssh.Client{}
	for i, remote := range remoteAddrs {
		clients = append(clients, gs.Proxy(bastion, remote, cfgs[i]))
	}

	for i, remote := range remoteAddrs {
		go TailLog(
			remote,
			logFile,
			onlyNewLine,
			clients[i],
			lines,
		)
	}

	for {
		select {
		case l := <-lines:
			log.Print(l)
		case sigTerm := <-termChan:
			log.Printf("signal %d received, process tail terminated on remote servers", sigTerm)
			for _, cli := range clients {
				sess, _ := cli.NewSession()
				err := sess.Run("pkill tail")
				if err != nil {
					log.Println(err)
				}
				sess.Close()
			}
			return
		}
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
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	usr, _ := user.Current()
	sc := gs.NewConfig()
	remoteHosts := StringArray{}
	logFile := flag.String("log_file", "", "full path for log file to tail")
	onlyNewLine := flag.Bool("only_new_line", true, "only print new lines appended after current log file")
	flag.Var(&remoteHosts, "remote_address",
		"remote address including port, eg. foo.com:22, may be given multiple times")
	remoteUser := flag.String("remote_user", usr.Username, "username to login to remote hosts")
	sc.ParseCommandLine()

	cli, err := sc.NewSSHClient()
	if err != nil {
		log.Fatal(err)
	}
	defer cli.Close()

	done := make(chan bool)
	hosts := strings.Split(remoteHosts.String(), ",")
	MultiTail(cli, hosts, *remoteUser, *logFile, *onlyNewLine, done)
}
