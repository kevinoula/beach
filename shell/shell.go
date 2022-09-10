package shell

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"fmt"
	"golang.org/x/crypto/ssh"
	"io"
	"log"
	"os"
)

type SSH struct {
	Username string
	Password string
	Hostname string
	client   *ssh.Client
	session  *ssh.Session
	stdin    io.WriteCloser
	stdout   io.Reader
	stderr   io.Reader
}

func (s *SSH) CreateSession() error {
	decodedPass, _ := base64.StdEncoding.DecodeString(s.Password)
	trimmedPass := bytes.TrimSpace(decodedPass)

	sshConfig := &ssh.ClientConfig{
		User: s.Username,
		Auth: []ssh.AuthMethod{ssh.Password(string(trimmedPass))},
	}

	sshConfig.HostKeyCallback = ssh.InsecureIgnoreHostKey()
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

func (s *SSH) StartSession() error {
	log.Printf("Attempting to SSH into %s...\n", s.Hostname)
	err := s.CreateSession()
	if err != nil {
		return fmt.Errorf("error connecting to host: %v\n", err)
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
		log.Fatalf("error connecting stdin to pipe: %v\n", err)
	}

	s.stdout, err = s.session.StdoutPipe()
	if err != nil {
		log.Fatalf("error connecting stdout to pipe: %v\n", err)
	}

	s.stderr, err = s.session.StderrPipe()
	if err != nil {
		log.Fatalf("error connecting stdout to pipe: %v\n", err)
	}

	// go routine to pass stdin to shell stdin
	wr := make(chan []byte, 10)
	go func() {
		for {
			select {
			case d := <-wr:
				_, err := s.stdin.Write(d)
				if err != nil {
					fmt.Printf("error writing to stdin: %v\n", err)
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
				fmt.Printf("error scanning: %v\n", scanner.Err())
			} else {
				fmt.Printf("\nio.EOF")
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

	return nil
}
