package state

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v2"
)

type ConnectionState map[string]bool

func Save(i string, s ConnectionState) {
	f, err := os.OpenFile("/etc/hyprspace/"+i+".state", os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("[!] Couldn't save state. Error: %v", err)
		return
	}
	defer f.Close()

	err = f.Truncate(0)
	if err != nil {
		fmt.Printf("[!] Couldn't truncate state fie. Error: %v", err)
		return
	}
	_, err = f.Seek(0, 0)
	if err != nil {
		fmt.Printf("[!] Couldn't save state. Error: %v", err)
		return
	}

	b, _ := yaml.Marshal(s)
	if _, err := f.Write(b); err != nil {
		fmt.Printf("[!] Couldn't save state. Error: %v", err)
		return
	}
}

func Read(i string) (st ConnectionState, err error) {
	s, err := os.ReadFile("/etc/hyprspace/" + i + ".state")
	if err != nil {
		return
	}

	err = yaml.Unmarshal(s, &st)
	return
}

func CleanUp(i string) {
	os.Remove("/etc/hyprspace/" + i + ".state")
}
