package gossh

import (
	"log"
	"os"
	"os/user"
	"testing"
)

func TestBasicSSH(t *testing.T) {
	usr, err := user.Current()
	if err != nil {
		log.Println(err)
	}
	log.Println(usr.Username, usr.HomeDir+"/.ssh/id_rsa")

	sc := &SSHConfig{
		User:    usr.Username,
		Host:    "localhost",
		Port:    "22",
		KeyFile: usr.HomeDir + "/.ssh/id_rsa",
	}
	agent, err := sc.SSHAgent()
	if err != nil {
		log.Println(err)
	}

	keyPair, err := sc.KeyPair()
	if err != nil {
		log.Println(err)
	}

	client, err := sc.Connect(keyPair, agent)
	if err != nil {
		log.Println(err)
	}
	defer client.Close()

	sess, err := client.NewSession()
	if err != nil {
		log.Println(err)
	}
	defer sess.Close()

	sess.Stdout = os.Stdout
	err = sess.Run("echo \"Hello World\"")
	if err != nil {
		log.Println(err)
	}
}
