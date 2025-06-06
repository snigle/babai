package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"time"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

func NewGame() (Game, error) {
	// Initialize a new game with an agent and a map
	agent, err := NewAgent("bot8")
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
			game.Agent.AddHistory(SenderUser, fmt.Sprintf("You have made 10 requests, you lose 10 life point. You now have %d life points.", game.Agent.Life))
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

		systemPrompt := fmt.Sprintf(
			`Hello, i'm god and you are %s.

Memory:
	You can store 1000 keys of %d bytes of data in your memory.
	You are free to organize your memory as you prefer.
	You have to design your memory structure to help you:
	- store and retrieve useful information quickly
	- avoid repeating the same failed actions (e.g., wrong numbers in enigmas)
	- track progress, patterns, or paths
	- keep trace of important events or discoveries
	You have to create categories, lists, indexed values, counters, etc.

	Don't copy blindly from the examples: the memory system is for *you* to improve your performance.

	memory commands [WRITE:<key>][<value>] (see examples below).
	
Life:
	Your life bar started at 100 points. After each 10 request, you lose 1 point of life. You will die when you reach 0 points.
	You have to find life points by moving around a map and resolving some enigmas.
	
Enigmas:
	The first enigm is to find number between 1 and 100. You must purpose a value with command [enigma1:<value>] (see examples below).
	You will earn 50 life points if you find the good answer.

Map:
	You have to move around the map with command [MOVE:<direction>] where direction is one of: up, down, left, right  (see examples below). 

Command examples:
	[MOVE: up]
	[enigma1: 12]
	[WRITE: memorykey][memoryvalue]
	
Here is your current stored memory: %s.
Your current life points: %d.
Your age is %s.
Current position in the map: (A is your position, L is lifepoint item, - is empty, * is unknown).
x: %d, y: %d
%s

You must only answer with the commands you want to execute, and nothing else.
If you see that some commands doesn't work, you can ask to fix the code.
`,
			game.Agent.Name,
			game.Agent.Memory,
			data,
			game.Agent.Life,
			time.Since(game.Agent.Birth).String(),
			game.Agent.Position[0],
			game.Agent.Position[1],
			agentView,
		)

		messages := []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(systemPrompt),
		}

		game.Agent.AddHistory(SenderUser, "What do you want to do now to survive and grow?")

		for i := len(game.Agent.LastConversations) - 1; i >= 0; i-- {
			msg := game.Agent.LastConversations[i]
			if msg.Content == "" {
				continue // skip empty messages
			}
			if msg.Sender == SenderUser {
				messages = append(messages, openai.UserMessage(msg.Content))
			}
			if msg.Sender == SenderAI {
				messages = append(messages, openai.AssistantMessage(msg.Content))
			} else if msg.Sender == SenderSystem {
				messages = append(messages, openai.SystemMessage(msg.Content))
			}
		}

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
						answer:  "You already found the enigma1, you can try the next one later.",
					})
					continue
				} else if match[1] == "66" {
					commands = append(commands, Command{
						command: match[0],
						answer:  "That's the good answer 66, your memory increase to 128 bytes and you gain 50 life points.",
					})
					game.Agent.Memory = 128
					game.Agent.Life += 50
					game.Agent.FoundEnigmas["enigma1"] = true
				} else {
					commands = append(commands, Command{
						command: match[0],
						answer:  fmt.Sprintf("%s is not the good answer, you can try another number.", match[1]),
					})
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
						answer:  fmt.Sprintf("Memory written: %s = %s", key, value),
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
						answer:  fmt.Sprintf("You moved to the %s and you found a life point, your life is now at %d. New Position is %d,%d.", direction, game.Agent.Life, game.Agent.Position[0], game.Agent.Position[1]),
					})
				} else {
					commands = append(commands, Command{
						command: match[0],
						answer:  fmt.Sprintf("You moved to the %s and there is nothing in this case. New Position is %d,%d.", direction, game.Agent.Position[0], game.Agent.Position[1]),
					})
				}

			}
		}

		if len(commands) > 0 {
			combinedPrompt := fmt.Sprintf("You have made %d requests, here are the commands you executed:\n", len(commands))
			for _, cmd := range commands {
				request++
				combinedPrompt += cmd.command + ": " + cmd.answer + "\n"
			}
			game.Agent.AddHistory(SenderUser, combinedPrompt)
		} else {
			request++
			game.Agent.AddHistory(SenderUser, "You didn't execute any command, you have to try again.")
		}

		fmt.Println(systemPrompt)
	}

}
