package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/sashabaranov/go-openai"
)

type AIService struct {
	client *openai.Client
}

type GeneratedTask struct {
	Title       string     `json:"title"`
	Description string     `json:"description"`
	DueDate     *time.Time `json:"due_date"`
}

func NewAIService(apiKey string) *AIService {
	return &AIService{
		client: openai.NewClient(apiKey),
	}
}

// GenerateTasksFromText analyzes text and extracts tasks using OpenAI GPT
func (s *AIService) GenerateTasksFromText(ctx context.Context, text string) ([]GeneratedTask, error) {
	if s.client == nil {
		return nil, fmt.Errorf("OpenAI client not initialized")
	}

	currentTime := time.Now().Format("2006-01-02 15:04:05")
	prompt := fmt.Sprintf(`あなたはタスク抽出アシスタントです。以下のテキストから具体的なタスクを抽出してください。

現在時刻: %s

テキスト:
%s

以下のJSON形式で、抽出したタスクの配列を返してください:
[
  {
    "title": "タスクのタイトル（簡潔に）",
    "description": "タスクの詳細説明",
    "due_date": "期限（ISO8601形式、例: 2025-10-28T23:59:59Z）。期限が明示されていない場合はnull"
  }
]

注意事項:
- タスクが1つもない場合は空の配列 [] を返してください
- 期限は相対的な表現（「明日」「来週」など）を具体的な日時に変換してください
- due_dateは必ずISO8601形式の文字列、またはnullにしてください
- JSONのみを返し、説明文は含めないでください`, currentTime, text)

	resp, err := s.client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model: openai.GPT4o,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: prompt,
				},
			},
			Temperature: 0.3,
		},
	)

	if err != nil {
		return nil, fmt.Errorf("OpenAI API error: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no response from OpenAI")
	}

	content := resp.Choices[0].Message.Content

	var tasks []GeneratedTask
	if err := json.Unmarshal([]byte(content), &tasks); err != nil {
		return nil, fmt.Errorf("failed to parse AI response: %w (response: %s)", err, content)
	}

	return tasks, nil
}
