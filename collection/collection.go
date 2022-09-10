package collection

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/kevinoula/beach/log"
	"io/fs"
	"io/ioutil"
	"os"
	"regexp"
	"strconv"
	"strings"
)

type ShellCollection struct {
	Shells map[string]map[string]string `json:"shells"`
}

type Collection struct {
	ShellCollection ShellCollection
	FileName        string
	file            *os.File
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

type Credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Hostname string `json:"hostname"`
}

// AddShell takes some credentials and adds them to the existing collection of SSH configurations. Credentials consist
// of a username, password, and a hostname. The password is encoded.
func (c *Collection) AddShell(creds Credentials) {
	shellConfig := map[string]map[string]map[string]string{"shells": {}} // Init an empty shell map for JSON
	if len(c.ShellCollection.Shells) > 0 {                               // Add collected shells
		shellConfig["shells"] = c.ShellCollection.Shells
	}
	shellConfig["shells"][creds.Hostname] = map[string]string{ // Add the new credentials
		"username": creds.Username,
		"password": creds.Password,
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
	c.refreshCollection()
	shells := c.ShellCollection.Shells
	fmt.Println("\nCollected shells:")
	i := 0
	options := map[string]string{}
	for hostname, shell := range shells {
		fmt.Printf("(%d) %s @ %s\n", i, shell["username"], hostname)
		options[strconv.Itoa(i)] = hostname
		i++
	}

	fmt.Println("\nDo one of the following:")
	fmt.Println("* Enter an option to connect to that shell via SSH (i.e. 0)")
	fmt.Println("* Enter a new shell (format: username@hostname.com)")
	fmt.Println("* Type `exit` to leave the beach")
	fmt.Println()
	for { // Keep reading input until an acceptable input is received
		var in string
		_, _ = fmt.Scanf("%s", &in)
		matched, _ := regexp.MatchString(".*(@).*", in)
		if matched { // Connect to a new SSH session
			input := strings.Split(in, "@")
			newUsername, newHostname := input[0], input[1]
			log.Debug.Printf("Detected new inputs username %s and hostname %s.", newUsername, newHostname)

			var newPassword string
			fmt.Println("Enter a password:")
			_, _ = fmt.Scanf("%s", &newPassword)
			encodedPass := base64.StdEncoding.EncodeToString([]byte(newPassword)) // Passwords should always be encoded
			newCreds := Credentials{Hostname: newHostname, Username: newUsername, Password: encodedPass}
			c.AddShell(newCreds)
			// TODO start SSH session

		} else if hostname, found := options[in]; found { // Connect to a previously stored SSH session
			fmt.Printf("Logging onto %s\n", hostname)
			// TODO start SSH session

		} else if strings.ToLower(in) == "exit" { // Exit the routine
			fmt.Println("Exiting...")
			return

		} else {
			fmt.Printf("Unrecognized input: %s\n", in)
		}
	}

}