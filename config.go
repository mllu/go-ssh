package gossh

import (
	"flag"
	"log"
	"os/user"

	"golang.org/x/crypto/ssh"
)

/*
 user (string) - ssh login name
 addr (string) - server hostname with port for ssh
 key (string) - path of private key for ssh
 pwd (string) - (optional) password to login
*/
type SSHConfig struct {
	User     string
	Host     string
	Port     string
	KeyFile  string
	Password string
	Client   *ssh.Client
}

func NewConfig() *SSHConfig {
	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}
	return &SSHConfig{
		User:    usr.Username,
		Host:    "localhost",
		Port:    "22",
		KeyFile: usr.HomeDir + "/.ssh/id_rsa",
	}
}

func (sc *SSHConfig) ParseCommandLine() {
	flag.StringVar(&sc.Host, "login_host", sc.Host, "hostname to ssh")
	flag.StringVar(&sc.Port, "login_port", sc.Port, "port to ssh")
	flag.StringVar(&sc.User, "login_user", sc.User, "username to ssh login")
	flag.StringVar(&sc.KeyFile, "key_file", sc.KeyFile, "location of private key")
	flag.Parse()

}
