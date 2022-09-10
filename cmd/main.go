package main

import (
	"bufio"
	"fmt"
	"golang.org/x/crypto/ssh"
	"io"
	"log"
	"os"
)

var (
	hostname string
	username string
	port     string
)

func connectToHost(user, host string) (*ssh.Client, *ssh.Session, error) {
	var pass string
	fmt.Print("Password: ")
	fmt.Scanf("%s\n", &pass)

	sshConfig := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{ssh.Password(pass)},
	}

	sshConfig.HostKeyCallback = ssh.InsecureIgnoreHostKey()
	client, err := ssh.Dial("tcp", host, sshConfig)
	if err != nil {
		return nil, nil, err
	}

	session, err := client.NewSession()
	if err != nil {
		client.Close()
		return nil, nil, err
	}

	return client, session, nil
}

func main() {
	hostname = ""
	username = ""
	port = "22"

	log.Printf("Attempting to SSH into %s...\n", hostname)
	c, s, err := connectToHost(username, hostname+":"+port)
	defer c.Close()
	if err != nil {
		log.Fatalf("error connecting to host: %v\n", err)
	}

	var stdin io.WriteCloser
	var stdout, stderr io.Reader
	defer s.Close()

	stdin, err = s.StdinPipe()
	if err != nil {
		log.Fatalf("error connecting stdin to pipe: %v\n", err)
	}

	stdout, err = s.StdoutPipe()
	if err != nil {
		log.Fatalf("error connecting stdout to pipe: %v\n", err)
	}

	stderr, err = s.StderrPipe()
	if err != nil {
		log.Fatalf("error connecting stdout to pipe: %v\n", err)
	}

	// go routine to pass stdin to shell stdin
	wr := make(chan []byte, 10)
	go func() {
		for {
			select {
			case d := <-wr:
				_, err := stdin.Write(d)
				if err != nil {
					fmt.Printf("error writing to stdin: %v\n", err)
					break
				}
			}
		}

	}()

	// go routine to scan shell stdout
	go func() {
		scanner := bufio.NewScanner(stdout)
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
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			fmt.Println(scanner.Text())
		}
	}()

	// Open SSH session
	s.Shell()

	for {

		fmt.Printf("%s@%s $ ", username, hostname)
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		text := scanner.Text()
		if text == "exit" {
			return
		}
		wr <- []byte(text + "\n")

	}

	return
}
