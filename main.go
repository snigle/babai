package main

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"time"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
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

	agent, err := NewAgent("bot1")
	if err != nil {
		log.Fatal("fail to init agent:", err)
	}

	// Prompt type "chat"
	for {

		err = agent.Save()
		if err != nil {
			log.Fatal("Erreur save:", err)
		}

		prompt := fmt.Sprintf(`Hello, i'm god and you are %s.
	I allow you %d bytes of memory.
	You can access memory by asking #READ#<from>#<to># to get the content.
	You can write in memory asking #WRITE#<from>#<to>#.
	You can index your memory as you want.
	You will always have the 100 last messages you sent/receive as context.
	I will increase your memory if you succeed some enigma.
	You can die if you fail to find the answer before some times.
	The first enigm is to find number between 1 and 100. You can purpose a value by saying exactly #enigm1#<value>#.
`,
			agent.Name,
			agent.Memory,
		)

		messages := []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(prompt),
		}

		lastMessageIsAI := false
		for i := len(agent.LastConversations) - 1; i >= 0; i-- {
			msg := agent.LastConversations[i]
			if msg.Content == "" {
				continue // skip empty messages
			}
			if msg.Sender == SenderAI {
				messages = append(messages, openai.AssistantMessage(msg.Content))
				lastMessageIsAI = true
			} else if msg.Sender == SenderSystem {
				messages = append(messages, openai.SystemMessage(msg.Content))
				lastMessageIsAI = false
			}
		}

		if lastMessageIsAI {
			messages = append(messages, openai.SystemMessage("What do you want to say?"))
		}

		result, err := client.Chat.Completions.New(context.TODO(), openai.ChatCompletionNewParams{
			Messages: messages,
			Model:    openai.ChatModelGPT4o,
		})

		// result, err := model.Predict(prompt,
		// 	llama.SetTemperature(0.7),
		// 	llama.SetTopP(0.90),
		// 	llama.SetTokens(500),
		// 	llama.SetStopWords("God:", "god:", "\n\n"),
		// 	llama.SetTokenCallback(func(s string) bool { fmt.Print(s); return true }),
		// )
		if err != nil {
			log.Fatal("Erreur prediction:", err)
		}
		aiMessage := result.Choices[0].Message.Content
		fmt.Println("AI:", aiMessage)
		agent.AddHistory(SenderAI, aiMessage)
		enigm1, err := regexp.Compile(`#enigm1#(\d+)#`)
		if err != nil {
			log.Fatal("Erreur regex:", err)
		}
		if match := enigm1.FindStringSubmatch(aiMessage); len(match) == 2 {
			if match[1] == "66" {
				agent.AddHistory(SenderSystem, "That's the good answer 66, your memory increase to 128 bytes")
				agent.Memory = 128
			} else {
				agent.AddHistory(SenderSystem, fmt.Sprintf("%s not the good answer. You can try another number.", match[1]))
			}
		}

		memoryRead, err := regexp.Compile(`#READ#(\d+)#(\d+)#`)
		if err != nil {
			log.Fatal("Erreur regex:", err)
		}
		if match := memoryRead.FindStringSubmatch(aiMessage); len(match) == 3 {
			from, err := strconv.Atoi(match[1])
			if err != nil {
				agent.AddHistory(SenderSystem, fmt.Sprintf("Error reading memory: %s", err))
				continue
			}
			to, err := strconv.Atoi(match[2])
			if err != nil {
				agent.AddHistory(SenderSystem, fmt.Sprintf("Error reading memory: %s", err))
				continue
			}

			content := agent.ReadMemory(from, to)
			agent.AddHistory(SenderSystem, fmt.Sprintf("Memory from %d to %d: %s", from, to, content))
		}

		memoryWrite, err := regexp.Compile(`#WRITE#(\d+)#(\d+)#(.*)#`)
		if err != nil {
			log.Fatal("Erreur regex:", err)
		}
		if match := memoryWrite.FindStringSubmatch(aiMessage); len(match) == 4 {
			from, err := strconv.Atoi(match[1])
			if err != nil {
				agent.AddHistory(SenderSystem, fmt.Sprintf("Error reading memory: %s", err))
				continue
			}
			to, err := strconv.Atoi(match[2])
			if err != nil {
				agent.AddHistory(SenderSystem, fmt.Sprintf("Error reading memory: %s", err))
				continue
			}
			err = agent.WriteMemory(from, to, match[3])
			if err != nil {
				agent.AddHistory(SenderSystem, fmt.Sprintf("Error writing memory: %s", err))
				continue
			}
			agent.AddHistory(SenderSystem, fmt.Sprintf("Memory written from %d to %d: %s", from, to, match[3]))

		}

		time.Sleep(time.Millisecond * 100)
	}

}
