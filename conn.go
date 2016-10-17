package gossh

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/user"
	"strings"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

// Ref. 1: http://talks.rodaine.com/gosf-ssh/present.slide
// Ref. 2: http://blog.ralch.com/tutorial/golang-ssh-connection/

// given SSHConfig return a ssh client session
// remember to close session when finish communicating
func (sc *SSHConfig) NewSSHClient() (*ssh.Client, error) {
	if sc.Client != nil {
		return sc.Client, nil
	}
	keyPair, err := sc.KeyPair()
	if err != nil {
		return nil, err
	}

	agent, err := sc.SSHAgent()
	if err != nil {
		return nil, err
	}

	client, err := sc.Connect(keyPair, agent)
	if err != nil {
		return nil, err
	}
	// don't close ssh.Client here, let gc handle it
	//defer client.Close()
	return client, nil
}

func (sc *SSHConfig) NewSSHClientSession() (sess *ssh.Session, err error) {
	if sc.Client == nil {
		if sc.Client, err = sc.NewSSHClient(); err != nil {
			return nil, err
		}
	}
	sess, err = sc.Client.NewSession()
	if err != nil {
		return nil, err
	}
	return sess, nil
}

// Decompose SSH certificate file and store into system
// Ref: https://godoc.org/golang.org/x/crypto/ssh#PublicKeys
func (sc *SSHConfig) KeyPair() (ssh.AuthMethod, error) {
	pem, err := ioutil.ReadFile(sc.KeyFile)
	if err != nil {
		return nil, err
	}

	key, err := ssh.ParsePrivateKey(pem)
	if err != nil {
		return nil, err
	}

	return ssh.PublicKeys(key), nil
}

// Obtain stored private keys from system environment variable
func (sc *SSHConfig) SSHAgent() (ssh.AuthMethod, error) {
	agentSock, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK"))
	if err != nil {
		return nil, err
	}
	defer agentSock.Close()
	return ssh.PublicKeysCallback(agent.NewClient(agentSock).Signers), nil
}

func (sc *SSHConfig) GetAuthMethods() ([]ssh.AuthMethod, error) {
	auths := []ssh.AuthMethod{}
	keyPair, err := sc.KeyPair()
	if err != nil {
		return nil, err
	}
	auths = append(auths, keyPair)
	agent, err := sc.SSHAgent()
	if err != nil {
		return nil, err
	}
	auths = append(auths, agent)
	return auths, nil
}

func (sc *SSHConfig) Connect(authMethods ...ssh.AuthMethod) (*ssh.Client, error) {
	cfg := ssh.ClientConfig{
		User: sc.User,
		Auth: authMethods,
	}
	return ssh.Dial("tcp", sc.Host+":"+sc.Port, &cfg)
}

func Proxy(bastion *ssh.Client, addr string, cliCfg *ssh.ClientConfig) *ssh.Client {
	netConn, _ := bastion.Dial("tcp", addr)
	conn, chans, reqs, _ := ssh.NewClientConn(netConn, addr, cliCfg)
	return ssh.NewClient(conn, chans, reqs)
}

func GetMultiCliConfig(remoteAddrs []string, remoteUser string) ([]*ssh.ClientConfig, error) {
	cfgs := []*ssh.ClientConfig{}
	usr, _ := user.Current()
	for _, remote := range remoteAddrs {
		part := strings.Split(remote, ":")
		if len(part) != 2 {
			return nil, fmt.Errorf("malformed remoteAddrs %s", remote)
		}
		host, port := part[0], part[1]
		sc := &SSHConfig{
			User:    remoteUser,
			Host:    host,
			Port:    port,
			KeyFile: usr.HomeDir + "/.ssh/id_rsa",
		}
		auths, err := sc.GetAuthMethods()
		if err != nil {
			return nil, err
		}
		cfg := &ssh.ClientConfig{
			User: sc.User,
			Auth: auths,
		}
		cfgs = append(cfgs, cfg)
	}
	return cfgs, nil
}
