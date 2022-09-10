package collection

import (
	"fmt"
	"github.com/kevinoula/beach/log"
	"io/fs"
	"os"
)

type Collection struct {
	file *os.File
}

type Credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Hostname string `json:"hostname"`
}

// InitCollection initializes an existing or new collection
func InitCollection() Collection {
	h, _ := os.UserHomeDir()
	c := fmt.Sprintf("%s/.beach-collection.json", h)
	newColl := Collection{}
	err := newColl.RetrieveOrCreateFile(c)
	if err != nil {
		log.Err.Fatalf("error generating collection file: %v", err)
	}
	return newColl
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
