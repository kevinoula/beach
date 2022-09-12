package shell

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"fmt"
	log "github.com/kevinoula/beach/log"
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

	// stderr is the IO reader for the SSH session which reads errors from the remote server output
	stderr io.Reader
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

	for {
		fmt.Printf("%s@%s $ ", s.Username, s.Hostname)
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		text := scanner.Text()
		if text == "exit" {
			return nil
		}
		wr <- []byte(text + "\n")
	}
}
