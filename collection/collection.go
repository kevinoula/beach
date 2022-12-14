package collection

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/kevinoula/beach/log"
	"github.com/kevinoula/beach/shell"
	"golang.org/x/crypto/ssh/terminal"
	"io/fs"
	"io/ioutil"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// ShellCollection is a stored collection of shell configurations.
type ShellCollection struct {
	// Shells is a map containing previously stored SSH credentials.
	Shells map[string]map[string]string `json:"shells"`
}

// Collection is a general object that stores a collection of shells and the source-of-truth file for the SSH
// configurations.
type Collection struct {
	// ShellCollection is a reference to a Collection's store of SSH configuration map.
	ShellCollection ShellCollection

	// FileName is a reference to the name of the file which serves as the source-of-truth for all SSH configurations
	FileName string

	// file is the file object that is created and read using the OS library.
	file *os.File
}

// RetrieveOrCreateFile looks to see if the file already exists. If it does not, it creates it as an empty
// file. The file is returned.
func (c *Collection) RetrieveOrCreateFile(path string) error {
	credsFile, err := os.Open(path)
	switch err {
	case nil:
		log.Debug.Printf("Found %s\n", path)

	case err.(*fs.PathError):
		log.Warn.Printf("Creating %s since it does not exist.", path)
		credsFile, err = os.Create(path)
		if err != nil {
			log.Err.Fatalf("Error creating collection file: %v", err)
		}
		log.Info.Printf("Initialized collection!\n")

	default:
		return fmt.Errorf("unable to read %s. Error: %v", path, err)
	}

	c.file = credsFile
	return nil
}

// refreshCollection is used to update the Collection's store of SSH sessions by reading from the stored file.
func (c *Collection) refreshCollection() {
	file, err := os.ReadFile(c.FileName)
	if err != nil {
		log.Err.Printf("Error reading %s: %v\n", c.FileName, err)
		return
	}

	var shells ShellCollection
	_ = json.Unmarshal(file, &shells)
	log.Debug.Printf("File contents: %v\n", string(file))
	log.Debug.Printf("Shells collected: %v\n", shells)
	c.ShellCollection = shells
}

// InitCollection initializes an existing or new collection with a stored file and an up-to-date collection
// of SSH sessions.
func InitCollection() Collection {
	h, _ := os.UserHomeDir()
	c := fmt.Sprintf("%s/.beach-shells.json", h)
	newColl := Collection{FileName: c}
	err := newColl.RetrieveOrCreateFile(c)
	if err != nil {
		log.Err.Fatalf("error generating collection file: %v", err)
	}
	return newColl
}

// AddShell takes some credentials and adds them to the existing collection of SSH configurations. Credentials consist
// of a username, password, and a hostname. The password is encoded.
func (c *Collection) AddShell(ssh shell.SSH) {
	shellConfig := map[string]map[string]map[string]string{"shells": {}} // Init an empty shell map for JSON
	if len(c.ShellCollection.Shells) > 0 {                               // Add collected shells
		shellConfig["shells"] = c.ShellCollection.Shells
	}
	key := fmt.Sprintf("%s@%s", ssh.Username, ssh.Hostname)
	shellConfig["shells"][key] = map[string]string{ // Add the new credentials
		"username": ssh.Username,
		"password": ssh.Password,
		"hostname": ssh.Hostname,
	}

	jsonStr, err := json.Marshal(shellConfig)
	if err != nil {
		log.Debug.Printf("Shell config used: %v\n", shellConfig)
		log.Err.Fatalf("Unable to convert configs to JSON: %v", err)
	}

	err = ioutil.WriteFile(c.FileName, jsonStr, 0644) // This will overwrite the existing file
	if err != nil {
		log.Debug.Printf("JSON string: %v\n", string(jsonStr))
		log.Err.Fatalf("Failed to write configs to collection: %v", err)
	}

	c.refreshCollection()
}

// DisplayShellAndOptions is the main user interface where the user can select various inputs to perform actions. The
// options include connecting to a previously stored session, connecting to a new SSH session, and exiting.
func (c Collection) DisplayShellAndOptions() {
	for { // Keep reading input until an acceptable input is received
		c.refreshCollection()
		shells := c.ShellCollection.Shells
		fmt.Println("\nCollected shells:")
		i := 0
		options := map[string]string{} // i.e. maps `0` to `username@hostname.com`
		for connectionName := range shells {
			fmt.Printf("(%d) %s\n", i, connectionName)
			options[strconv.Itoa(i)] = connectionName
			i++
		}

		fmt.Println("\nDo one of the following:")
		fmt.Println("* Enter an option to connect to that sh via SSH (i.e. 0)")
		fmt.Println("* Enter a new sh (format: username@hostname.com)")
		fmt.Println("* Type `exit` to leave the beach")
		fmt.Println()

		var in string
		fmt.Printf("$ ")
		_, _ = fmt.Scan(&in)
		matched, _ := regexp.MatchString(".*(@).*", in)
		if matched { // Connect to a new SSH session
			input := strings.Split(in, "@")
			newUsername, newHostname := input[0], input[1]
			log.Debug.Printf("Detected new inputs username %s and hostname %s.", newUsername, newHostname)
			if len(newUsername) == 0 || len(newHostname) == 0 {
				log.Warn.Println("Invalid username@hostname.com input.")
				continue
			}

			fmt.Println("Enter a password:")
			newPassword, _ := terminal.ReadPassword(0)
			encodedPass := base64.StdEncoding.EncodeToString(newPassword) // Passwords should always be encoded

			newSSH := shell.SSH{Hostname: newHostname, Username: newUsername, Password: encodedPass}
			err := newSSH.StartSession()
			if err != nil { // Any errors will not add the shell to the collection
				log.Err.Printf("Error starting session: %v\n", err)
				continue
			}
			c.AddShell(newSSH)

		} else if connectionName, found := options[in]; found { // Connect to a previously stored SSH session
			newSSH := shell.SSH{Hostname: shells[connectionName]["hostname"], Username: shells[connectionName]["username"], Password: shells[connectionName]["password"]}
			err := newSSH.StartSession()
			if err != nil {
				log.Err.Printf("Error starting session: %v\n", err)
			}

		} else if strings.ToLower(in) == "exit" { // Exit the routine
			fmt.Println("Exiting...")
			return

		} else {
			fmt.Printf("Unrecognized input: %s\n", in)
		}
	}

}
