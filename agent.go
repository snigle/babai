package main

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

const MaxKeyNumber = 1000

type Agent struct {
	Name              string
	Memory            int
	Life              uint
	Data              map[string]string
	LastConversations []Message
	Position          [2]int          // Position in the map, e.g. [x, y]
	FoundEnigmas      map[string]bool // List of enigmas found by the agent
}

func (a *Agent) WriteMemory(key, value string) error {
	if a.Data == nil {
		a.Data = make(map[string]string)
	}

	if value == "" {
		delete(a.Data, key)
		return nil
	}
	if len(a.Data) > 100 {
		return fmt.Errorf("memory is full, maximum %d keys allowed", MaxKeyNumber)
	}

	if len(value) > a.Memory {
		return fmt.Errorf("value '%s' is too long, maximum %d characters", value, a.Memory)
	}

	a.Data[key] = value

	return nil
}

type Message struct {
	Sender  Sender
	Content string
}

func NewAgent(name string) (Agent, error) {
	res := Agent{Name: name}
	err := res.Load()
	return res, err
}

func (a *Agent) Load() error {
	filename := fmt.Sprintf("%s.yaml", a.Name)

	// Vérifie si le fichier existe
	_, err := os.Stat(filename)
	if os.IsNotExist(err) {
		// Fichier inexistant : on crée un fichier YAML vide (avec les valeurs par défaut)
		a.Memory = 56
		a.Life = 100
		a.LastConversations = make([]Message, 10) // Initialisation avec une taille fixe
		a.FoundEnigmas = make(map[string]bool)
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
	SenderUser
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
