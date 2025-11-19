package resilience

type DeadLetterQueue struct {
	messages []string
}

func NewDeadLetterQueue() *DeadLetterQueue {
	return &DeadLetterQueue{
		messages: make([]string, 0),
	}
}
func (dlq *DeadLetterQueue) Add(message string) {
	dlq.messages = append(dlq.messages, message)
}

func (dlq *DeadLetterQueue) GetMessages() []string {
	return dlq.messages
}

func ProcessWithDLQ(messages []string, processor func(string) error, dlq *DeadLetterQueue) {
	for _, msg := range messages {
		err := processor(msg)
		if err != nil {
			dlq.Add(msg)
		}
	}
}
