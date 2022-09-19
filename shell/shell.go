package shell

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"fmt"
	"github.com/kevinoula/beach/log"
	"golang.org/x/crypto/ssh"
	"io"
	"os"
	"time"
)

// SSH is an object which runs a single SSH session for a given user.
type SSH struct {
	// Username is the username provided to sign onto an SSH session.
	Username string

	// Password is the username provided to sign onto an SSH session.
	Password string

	// Host is the username provided to sign onto an SSH session.
	Hostname string

	// client is the generated SSH client that handles the TLS handshake between the user and remote server.
	client *ssh.Client

	// session is the generated SSH session that delivers user input and remote server output.
	session *ssh.Session

	// stdin is the IO writer for the SSH session where the user sends inputs to.
	stdin io.WriteCloser

	// stdout is the IO reader for the SSH session which reads the remote server output.
	stdout io.Reader

	// stderr is the IO reader for the SSH session which reads errors from the remote server output.
	stderr io.Reader

	// cmdHistory is a stack containing the user's recent cmd history.
	cmdHistory []string
}

// addCmdToStack adds a user provided cmd input into the SSH's command history.
func (s *SSH) addCmdToStack(cmd string) {
	// Store only the last 5 commands
	if len(s.cmdHistory) > 4 {
		s.cmdHistory = s.cmdHistory[1:]
	}
	s.cmdHistory = append(s.cmdHistory, cmd)
}

// displayCmdHistory prints out the user's input history for an ongoing SSH session.
func (s *SSH) displayCmdHistory() {
	// Print all entered cmd from the top to bottom of the stack
	c := 1
	fmt.Println("User cmd history:")
	for i := len(s.cmdHistory); i > 0; i-- {
		fmt.Printf("(%d) %s\n", c, s.cmdHistory[i-1])
		c++
	}
}

// CreateSession creates all the necessary components to begin an SSH session with a remote server.
func (s *SSH) CreateSession() error {
	decodedPass, _ := base64.StdEncoding.DecodeString(s.Password)
	trimmedPass := bytes.TrimSpace(decodedPass)

	sshConfig := &ssh.ClientConfig{
		User:    s.Username,
		Auth:    []ssh.AuthMethod{ssh.Password(string(trimmedPass))},
		Timeout: time.Second * 10,
	}

	sshConfig.HostKeyCallback = ssh.InsecureIgnoreHostKey() // TODO validate host key
	client, err := ssh.Dial("tcp", s.Hostname+":22", sshConfig)
	if err != nil {
		return err
	}

	session, err := client.NewSession()
	if err != nil {
		_ = client.Close()
		return err
	}

	s.client = client
	s.session = session
	return nil
}

// StartSession uses the SSH client and session to begin serving input and outputs from the user to the remote server and back.
func (s *SSH) StartSession() error {
	log.Info.Printf("Attempting to SSH into %s@%s...\n", s.Username, s.Hostname)
	err := s.CreateSession()
	if err != nil {
		return fmt.Errorf("creating session resulted in %v\n", err)
	}

	// Defer closing client and session
	defer func(client *ssh.Client) {
		_ = client.Close()
	}(s.client)

	defer func(session *ssh.Session) {
		_ = session.Close()
	}(s.session)

	s.stdin, err = s.session.StdinPipe()
	if err != nil {
		return fmt.Errorf("connecting stdin to pipe resulted in %v\n", err)
	}

	s.stdout, err = s.session.StdoutPipe()
	if err != nil {
		return fmt.Errorf("connecting stdout to pipe resulted in %v\n", err)
	}

	s.stderr, err = s.session.StderrPipe()
	if err != nil {
		return fmt.Errorf("connecting stdout to pipe resulted in %v\n", err)
	}

	// go routine to pass stdin to shell stdin
	wr := make(chan []byte, 10)
	go func() {
		for {
			select {
			case d := <-wr:
				_, err := s.stdin.Write(d)
				if err != nil {
					log.Err.Printf("writing to stdin resulted in %v\n", err)
					break
				}
			}
		}

	}()

	// go routine to scan shell stdout
	go func() {
		scanner := bufio.NewScanner(s.stdout)
		for {
			if tkn := scanner.Scan(); tkn {
				rcv := scanner.Bytes()
				raw := make([]byte, len(rcv))
				copy(raw, rcv)
				fmt.Println(string(raw))
			} else if scanner.Err() != nil {
				log.Err.Printf("error scanning: %v\n", scanner.Err())
			} else {
				log.Err.Println("io.EOF")
				break
			}
		}
	}()

	// go routine to scan stderr
	go func() {
		scanner := bufio.NewScanner(s.stderr)
		for scanner.Scan() {
			fmt.Println(scanner.Text())
		}
	}()

	// Open SSH session
	_ = s.session.Shell()
	fmt.Printf("Connected to %s\n", s.Hostname)
	fmt.Println("* Enter `hist` to see user cmd history")
	fmt.Println("* Enter `exit` to end SSH session")
	for {
		fmt.Printf("%s@%s $ ", s.Username, s.Hostname)
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		text := scanner.Text()
		switch text {
		case "hist":
			s.displayCmdHistory()

		case "exit":
			return nil

		default:
			s.addCmdToStack(text)
			wr <- []byte(text + "\n")
			time.Sleep(time.Second * 1) // short input delay to allow output to populate
		}
	}
}
