package errors

import (
	"sync"
	"time"
)

// TUIHandler handles errors by storing them for display in the TUI.
type TUIHandler struct {
	mu       sync.RWMutex
	messages []Message
	onError  func(msg Message)
}

type Message struct {
	Text      string
	Type      MessageType
	Timestamp time.Time
}

type MessageType int

const (
	MessageTypeError MessageType = iota
	MessageTypeWarning
	MessageTypeInfo
	MessageTypeSuccess
)

func NewTUIHandler(onError func(msg Message)) *TUIHandler {
	return &TUIHandler{
		messages: make([]Message, 0),
		onError:  onError,
	}
}

func (h *TUIHandler) Error(msg string) {
	h.addMessage(msg, MessageTypeError)
}

func (h *TUIHandler) Warning(msg string) {
	h.addMessage(msg, MessageTypeWarning)
}

func (h *TUIHandler) Info(msg string) {
	h.addMessage(msg, MessageTypeInfo)
}

func (h *TUIHandler) Success(msg string) {
	h.addMessage(msg, MessageTypeSuccess)
}

func (h *TUIHandler) addMessage(msg string, msgType MessageType) {
	h.mu.Lock()
	defer h.mu.Unlock()

	message := Message{
		Text:      msg,
		Type:      msgType,
		Timestamp: time.Now(),
	}
	h.messages = append(h.messages, message)

	if h.onError != nil {
		h.onError(message)
	}
}

func (h *TUIHandler) GetLatest() (Message, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if len(h.messages) == 0 {
		return Message{}, false
	}
	return h.messages[len(h.messages)-1], true
}

func (h *TUIHandler) Clear() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.messages = make([]Message, 0)
}

func (h *TUIHandler) GetAll() []Message {
	h.mu.RLock()
	defer h.mu.RUnlock()

	copied := make([]Message, len(h.messages))
	copy(copied, h.messages)
	return copied
}
