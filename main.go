package main

import (
	"encoding/json"
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"os"
	"runtime"
)

func main() {
	app := fiber.New(fiber.Config{
		Prefork:     true,
		Concurrency: runtime.NumCPU(),
	})
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept",
		MaxAge:       3000,
	}))

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	// 普通聊天完成接口
	app.Post("/v1/chat/completions", func(c *fiber.Ctx) error {
		//fmt.Printf("收到请求: %s %s\nBody: %s\n", c.Method(), c.Path(), c.Body())
		var req ChatCompletionRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}

		// 调用VLLM模型处理请求
		if req.Stream == true {
			c.Set("Content-Type", "text/event-stream")
			c.Set("Cache-Control", "no-cache")
			c.Set("Connection", "keep-alive")
			return processStreamingResponse(c, &req)
		}
		resp, err := processChatCompletion(&req)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(resp)
	})
	app.Listen(":3000")
}

type ChatCompletionRequest struct {
	Messages []struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"messages"`
	Stream bool `json:"stream"`
}

func processChatCompletion(req *ChatCompletionRequest) (interface{}, error) {
	fileContent, err := os.ReadFile("response.json")
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(fileContent, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func processStreamingResponse(c *fiber.Ctx, req *ChatCompletionRequest) error {
	//fmt.Printf("开始流式响应: %s %s\n", c.Method(), c.Path())
	fileContent, err := os.ReadFile("response.json")
	if err != nil {
		return err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(fileContent, &result); err != nil {
		return err
	}

	loopString := "嗯，用户刚刚说“你是一个助理”，然后我回应“你好”。现在用户又发了一个空的查询，可能是在等我继续说什么，或者他没有输入内容。我应该怎么回应呢？\n\n先看看用户的历史消息，他可能是在测试我是否能正确回应，或者是在闲聊。考虑到他刚才提到我是一个助理，可能是在确认我的身份。我可能需要确认自己能提供帮助，或者询问他具体需要什么帮助。\n\n但用户这次发送的是一个空的查询，可能是个错误或者他还没输入内容。我应该礼貌地询问他有什么可以帮助的，或者请他提供更多的信息。这样既不会显得呆板，又能引导用户继续对话。\n\n总的来说，我应该回复：“好的，请问有什么可以帮助您的？”这样既确认了我的角色，又邀请用户提出需求。\n\n\n好的，请问有什么可以帮助您的？"
	for _, char := range loopString {
		chunk := map[string]interface{}{
			"id":      "chatcmpl-8a20a3cf9ab24adb9527ea6f61831e31",
			"object":  "chat.completion.chunk",
			"created": 1744611565,
			"model":   "DeepSeek-R1",
			"choices": []map[string]interface{}{
				{
					"index": 0,
					"delta": map[string]interface{}{
						"content": string(char),
					},
					"logprobs":      nil,
					"finish_reason": nil,
				},
			},
		}
		jsonData, _ := json.Marshal(chunk)
		c.Write([]byte(fmt.Sprintf("data: %s\n\n", jsonData)))
		//time.Sleep(2 * time.Millisecond)
	}
	finalChunk := map[string]interface{}{
		"id":      "chatcmpl-8a20a3cf9ab24adb9527ea6f61831e31",
		"object":  "chat.completion.chunk",
		"created": 1744611565,
		"model":   "DeepSeek-R1",
		"choices": []map[string]interface{}{
			{
				"index": 0,
				"delta": map[string]interface{}{
					"content": "",
				},
				"logprobs":      nil,
				"finish_reason": "stop",
			},
		},
	}
	jsonData, _ := json.Marshal(finalChunk)
	c.Write([]byte(fmt.Sprintf("data: %s\n\n", jsonData)))
	c.Write([]byte("data: [DONE]\n\n"))
	return nil
}
