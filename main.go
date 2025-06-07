package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"time"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

func NewGame() (Game, error) {
	// Initialize a new game with an agent and a map
	agent, err := NewAgent("bot1")
	if err != nil {
		return Game{}, fmt.Errorf("failed to create agent: %w", err)
	}
	gameMap, err := LoadMap("map.json")
	if err != nil {
		return Game{}, fmt.Errorf("failed to load map: %w", err)
	}

	game := Game{
		Agent: agent,
		Map:   gameMap,
	}
	if agent.Position[0] == 0 && agent.Position[1] == 0 {
		game.Spawn()
	}
	return game, nil
}

const (
	MaxRequestNumber    = 10
	MaxConversationSize = 10
	StartLifePoints     = 100
	LostPointPerRequest = 10
	Enigma1Points       = 50
)

func main() {
	// Charge le modèle
	// model, err := llama.New("/home/pc/.cache/llama.cpp/NousResearch_Nous-Hermes-2-Mistral-7B-DPO-GGUF_Nous-Hermes-2-Mistral-7B-DPO.Q6_K.gguf", llama.EnableF16Memory, llama.SetContext(4096), llama.SetGPULayers(35))
	// if err != nil {
	// 	log.Fatal("Erreur chargement modèle:", err)
	// }

	client := openai.NewClient(
		option.WithAPIKey("My API Key"),                // defaults to os.LookupEnv("OPENAI_API_KEY")
		option.WithBaseURL("http://localhost:8080/v1"), // defaults to os.LookupEnv("OPENAI_API_BASE") or "https://api.openai.com/v1"
	)

	game, err := NewGame()
	if err != nil {
		log.Fatal("Erreur initialisation du jeu:", err)
	}

	// Prompt type "chat"
	request := 0
	for {

		if request >= 11 {
			if game.Agent.Life <= 10 {
				fmt.Println("Game Over! Your agent has no more life points.")
				break
			}
			game.Agent.Life -= 10
			request -= 10
			game.Agent.AddHistory(SenderUser, fmt.Sprintf("As I did more than 10 requests, I lost 10 life points. Now I have %d life points.", game.Agent.Life))
		}

		err = game.Agent.Save()
		if err != nil {
			log.Fatal("Erreur save:", err)
		}
		err = game.Map.Save()
		if err != nil {
			log.Fatal("Erreur save map:", err)
		}
		time.Sleep(time.Millisecond * 100)

		agentView := game.Map.GetAgentView(&game.Agent)
		data, err := json.Marshal(game.Agent.Data)
		if err != nil {
			log.Fatal("Erreur marshalling agent data:", err)
		}

		systemPrompt := `You are assistant which have to help IA to survive and grow in an artifial world. You have to help to to access the real world.

You will speak to another AI named %s. This is its capabilities:

Memory:
	He can store map of 1000 keys with ` + strconv.Itoa(game.Agent.Memory) + ` bytes of data in each value.
	He can organize its memory as he want.
	You must help him to structure it's memory to:
	- store and retrieve useful information quickly
	- avoid repeating the same failed actions (e.g., wrong numbers in enigmas)
	- track progress, patterns, or paths
	- keep trace of important events or discoveries

	To let him to write data in his memory, use this command: [WRITE:<key>][<value>] (see examples below).
	
Life:
	His life bar started at ` + strconv.Itoa(StartLifePoints) + ` points. After each ` + strconv.Itoa(MaxRequestNumber) + ` actions, He loose ` + strconv.Itoa(LostPointPerRequest) + ` point of life. He will die when he reach 0 points.
	You have to help him to find life points by moving around a map and resolving some enigmas.
	
Enigmas:
	The first enigma he have to resolve is to find number between 1 and 100. You can help him and purpose a value with command [enigma1:<number>] (see examples below).
	He will earn ` + strconv.Itoa(Enigma1Points) + ` life points if you find the good answer.

Map:
	He can move around the map with command [MOVE:<direction>] where direction is one of: up, down, left, right  (see examples below). 

Command examples:
	[MOVE: up]
	[enigma1: 12]
	[WRITE: agent.data][name=John Doe; age=30; actions=move;write;enigma1]
	`

		statePrompt := fmt.Sprintf(
			` Hello, I am %s, an AI agent in a virtual world.
My current stored memory is: %s.
My current life points: %d.
My age is %s.
The map around me: (A is my position, L is lifepoint item, - is empty, * is unknown).
%s
My position in the world: x: %d, y: %d

Which command I must use to survive now ?
`,
			game.Agent.Name,
			data,
			game.Agent.Life,
			time.Since(game.Agent.Birth).String(),
			agentView,
			game.Agent.Position[0],
			game.Agent.Position[1],
		)

		messages := []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(systemPrompt),
		}

		for i := len(game.Agent.LastConversations) - 1; i >= 0; i-- {
			msg := game.Agent.LastConversations[i]
			if msg.Content == "" {
				continue // skip empty messages
			}
			if msg.Sender == SenderUser {
				messages = append(messages, openai.UserMessage(msg.Content))
			}
			if msg.Sender == SenderAI {
				//messages = append(messages, openai.AssistantMessage(msg.Content))
			} else if msg.Sender == SenderSystem {
				messages = append(messages, openai.SystemMessage(msg.Content))
			}
		}

		messages = append(messages, openai.UserMessage(statePrompt))
		messages = append(messages, openai.SystemMessage("only answer with commands, do not write anything else."))

		result, err := client.Chat.Completions.New(context.TODO(), openai.ChatCompletionNewParams{
			Messages: messages,
			Model:    openai.ChatModelGPT4o,
		})

		if err != nil {
			log.Fatal("Erreur prediction:", err)
		}
		aiMessage := result.Choices[0].Message.Content

		game.Agent.AddHistory(SenderAI, aiMessage)
		enigma1, err := regexp.Compile(`(?i)\[enigma1: ?(\d+)\]`)
		if err != nil {
			log.Fatal("Erreur regex:", err)
		}
		allMatch := enigma1.FindAllStringSubmatch(aiMessage, -1)

		type Command struct {
			command string
			answer  string
		}
		commands := []Command{}
		for _, match := range allMatch {
			if len(match) == 2 {
				if game.Agent.FoundEnigmas["enigma1"] {
					commands = append(commands, Command{
						command: match[0],
						answer:  "I already found the enigma1, I will ask about next one later.",
					})
					continue
				} else if match[1] == "66" {
					commands = append(commands, Command{
						command: match[0],
						answer:  "That's the good answer 66, my memory increased to 128 bytes and I gain 50 life points.",
					})
					game.Agent.Memory = 128
					game.Agent.Life += 50
					game.Agent.FoundEnigmas["enigma1"] = true
				} else {
					commands = append(commands, Command{
						command: match[0],
						answer:  fmt.Sprintf("%s is not the good answer, should I try another number ?", match[1]),
					})
					game.Agent.WriteMemory("enigma1.failed_answer", game.Agent.Data["enigma1.failed_answer"]+";"+match[1])
				}
			}
		}
		memoryWrite, err := regexp.Compile(`(?i)\[WRITE: ?(.+?)\]\[(.*?)\]`)
		if err != nil {
			log.Fatal("Erreur regex:", err)
		}
		matchAll := memoryWrite.FindAllStringSubmatch(aiMessage, -1)
		for _, match := range matchAll {
			if len(match) == 3 {
				key := match[1]
				value := match[2]
				err = game.Agent.WriteMemory(key, value)
				if err != nil {
					commands = append(commands, Command{
						command: match[0],
						answer:  fmt.Sprintf("Error writing memory: %s", err.Error()),
					})
				} else {
					commands = append(commands, Command{
						command: match[0],
						answer:  fmt.Sprintf("Ok I wrote this in my memory: %s = %s", key, value),
					})
				}

			}
		}
		move, err := regexp.Compile(`(?i)\[MOVE: ?(up|down|left|right)\]`)
		if err != nil {
			log.Fatal("Erreur regex:", err)
		}
		for _, match := range move.FindAllStringSubmatch(aiMessage, -1) {
			if len(match) == 2 {
				direction := match[1]
				entity := game.MoveAgent(&game.Agent, direction)
				if entity.Type == EntityTypeLifePoint {
					commands = append(commands, Command{
						command: match[0],
						answer:  fmt.Sprintf("I moved %s and I found a life point! My life is now at %d. My new position is %d,%d.", direction, game.Agent.Life, game.Agent.Position[0], game.Agent.Position[1]),
					})
					game.Agent.WriteMemory("map.life_found", game.Agent.Data["map.life_found"]+";"+strconv.Itoa(game.Agent.Position[0])+"-"+strconv.Itoa(game.Agent.Position[1]))

				} else {
					commands = append(commands, Command{
						command: match[0],
						answer:  fmt.Sprintf("I moved %s and there is nothing in this case. My new position is %d,%d.", direction, game.Agent.Position[0], game.Agent.Position[1]),
					})
				}

			}
		}

		if len(commands) > 0 {
			combinedPrompt := fmt.Sprintf("You asked me to realize %d requests, here are the results:\n", len(commands))
			for _, cmd := range commands {
				request++
				combinedPrompt += cmd.command + ": " + cmd.answer + "\n"
			}
			game.Agent.AddHistory(SenderUser, combinedPrompt)
		} else {
			request++
			game.Agent.AddHistory(SenderUser, "I didn't understand what to do, I'm an AI and I only answer with commands.")
		}

		fmt.Println(statePrompt)
	}

}
