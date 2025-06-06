package main

import (
	"encoding/json"
	"math/rand"
	"os"
)

type Game struct {
	Agent Agent
	Map   Map
}

func (g *Game) MoveAgent(agent *Agent, direction string) Entity {

	switch direction {
	case "up":
		agent.Position[0]--
	case "down":
		agent.Position[0]++
	case "left":
		agent.Position[1]--
	case "right":
		agent.Position[1]++
	default:
		return Entity{}
	}

	if agent.Position[0] < 0 || agent.Position[0] >= len(g.Map) ||
		agent.Position[1] < 0 || agent.Position[1] >= len(g.Map[0]) {
		return Entity{}
	}

	if g.Map[agent.Position[0]][agent.Position[1]].Type == EntityTypeLifePoint {
		entity := g.Map[agent.Position[0]][agent.Position[1]]
		// If the agent moves to a life point, increase its life
		agent.Life += 15
		g.Map[agent.Position[0]][agent.Position[1]] = Entity{
			Type: EntityTypeEmpty, // Remove the life point from the map
			Name: "",
		}
		return entity
	}
	return Entity{}
}

type Map [1000][1000]Entity

type Entity struct {
	Name string
	Type EntityType
}

type EntityType int

const (
	EntityTypeEmpty EntityType = iota
	//	EntityTypeAI
	EntityTypeLifePoint
)

func GenerateMap() Map {
	var m Map
	for i := range m {
		for j := range m[i] {
			entityType := EntityTypeEmpty
			if (i+j)%(rand.Intn(5)+10) == 0 {
				entityType = EntityTypeLifePoint
			}
			// Initialize each cell with an empty entity
			m[i][j] = Entity{
				Type: entityType,
			}
		}
	}
	return m
}

func (g *Game) Spawn() {
	if g.Agent.Position[0] != 0 || g.Agent.Position[1] != 0 {
		// If the agent already has a position, we do not spawn it again
		return
	}

	// Spawn the agent at a random position in the map
	for {
		// Generate a random position within the bounds of the map
		i := rand.Intn(len(g.Map))
		j := rand.Intn(len(g.Map[i]))
		if g.Map[i][j].Type == EntityTypeEmpty {
			// If the position is empty, place the agent there
			g.Agent.Position[0] = i
			g.Agent.Position[1] = j
			return
		}
	}
}

func (m Map) GetAgentView(agent *Agent) string {
	agentView := 4
	var view string
	for i := agent.Position[0] - agentView; i <= agent.Position[0]+agentView; i++ {
		for j := agent.Position[1] - agentView; j <= agent.Position[1]+agentView; j++ {
			if i == agent.Position[0] && j == agent.Position[1] {
				view += "A " // Agent's position
				continue
			}
			// Check if the position is within the bounds of the map
			if i < 0 || i >= len(m) || j < 0 || j >= len(m[i]) {
				view += "- " // Out of bounds
				continue
			}
			switch m[i][j].Type {
			case EntityTypeEmpty:
				view += "- "
			case EntityTypeLifePoint:
				view += "L "
			}
		}
		view += "\n"
	}
	return view
}

func (g Game) String() string {
	var result string
	for i := range g.Map {
		for j := range g.Map[i] {
			if i == g.Agent.Position[0] && j == g.Agent.Position[1] {
				result += "A " // Agent's position
				continue
			}
			switch g.Map[i][j].Type {
			case EntityTypeEmpty:
				result += "- "
			case EntityTypeLifePoint:
				result += "L "
			}
		}
		result += "\n"
	}
	return result
}

func (m Map) Save() error {
	file, err := os.Create("map.json")
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)

	return encoder.Encode(m)
}

func LoadMap(filename string) (Map, error) {
	var m Map
	file, err := os.Open(filename)
	if err != nil {
		if os.IsNotExist(err) {
			// If the file does not exist, return an empty map
			return GenerateMap(), nil
		}
		return m, err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	err = decoder.Decode(&m)
	if err != nil {
		return m, err
	}
	return m, nil
}
