package main

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Agent struct {
	Name              string
	Memory            int
	Data              []byte
	LastConversations [100]Message
}

func (a Agent) WriteMemory(from int, to int, s string) error {
	if from < 0 || to >= a.Memory {
		return fmt.Errorf("Memory out of bounds; try to write a smaller range.")
	}
	if len(s) > to-from+1 {
		return fmt.Errorf("Data too long; try to write a shorter string.")
	}
	if len(s) == 0 {
		return fmt.Errorf("Empty data; try to write something first like your name or the solution of the first enigma.")
	}

	for i := from; i <= to; i++ {
		if i-from < len(s) {
			a.Data[i] = s[i-from]
		} else {
			a.Data[i] = 0 // Remplissage avec des zéros si la chaîne est plus courte que la plage
		}
	}
	return a.Save()
}

func (a Agent) ReadMemory(from int, to int) []byte {
	if from < 0 || to >= a.Memory {
		return []byte("Memory out of bounds; try to read a smaller range.")
	}
	data := a.Data[from:to]
	if string(data) == string(make([]byte, len(data))) {
		return []byte("No data in memory; try to write something first like your name or the solution of the first enigma.")
	}
	return data
}

type Message struct {
	Sender  Sender
	Content string
}

func NewAgent(name string) (Agent, error) {
	res := Agent{Name: name}
	err := res.Load()
	if len(res.Data) != res.Memory {
		tmp := make([]byte, res.Memory)
		for i, b := range res.Data {
			tmp[i] = b
		}
		res.Data = tmp
	}
	return res, err
}

func (a *Agent) Load() error {
	filename := fmt.Sprintf("%s.yaml", a.Name)

	// Vérifie si le fichier existe
	_, err := os.Stat(filename)
	if os.IsNotExist(err) {
		// Fichier inexistant : on crée un fichier YAML vide (avec les valeurs par défaut)
		a.Memory = 56
		return a.Save()
	} else if err != nil {
		return err // autre erreur
	}

	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	decoder := yaml.NewDecoder(file)
	err = decoder.Decode(a)
	if err != nil {
		return err
	}
	return nil
}

func (a *Agent) Save() error {

	filename := fmt.Sprintf("%s.yaml", a.Name)

	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := yaml.NewEncoder(file)
	defer encoder.Close()

	return encoder.Encode(a)

}

func (a *Agent) AddHistory(sender Sender, message string) {
	for i := len(a.LastConversations) - 1; i > 0; i-- {
		a.LastConversations[i] = a.LastConversations[i-1]
	}
	a.LastConversations[0] = Message{
		Sender:  sender,
		Content: message,
	}
}

type Sender int

const (
	SenderSystem Sender = iota
	SenderAI
	SenderGoogle
)

func (s Sender) String() string {
	switch s {
	case SenderSystem:
		return "system"
	case SenderAI:
		return "ai"
	case SenderGoogle:
		return "google"
	default:
		return "unknown"
	}
}
