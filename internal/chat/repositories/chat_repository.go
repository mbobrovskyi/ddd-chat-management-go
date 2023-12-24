package repositories

import (
	"github.com/mbobrovskyi/ddd-chat-management-go/internal/chat/domain"
	"github.com/mbobrovskyi/ddd-chat-management-go/internal/chat/domain/chat"
	"github.com/samber/lo"
	"time"
)

var _ domain.Repository = (*ChatRepository)(nil)

type ChatRepository struct {
	chats []chat.Chat
}

func (r *ChatRepository) getLastId() uint64 {
	return lo.MaxBy(r.chats, func(a chat.Chat, b chat.Chat) bool {
		return a.GetId() > b.GetId()
	}).GetId()
}

func (r *ChatRepository) GetAll() ([]chat.Chat, error) {
	return r.chats, nil
}

func (r *ChatRepository) GetById(id uint64) (chat.Chat, error) {
	for _, c := range r.chats {
		if c.GetId() == id {
			return chat.NewChat(c.GetId(), c.GetName(), c.GetType(), c.GetImage(), c.GetLastMessage(),
				c.GetCreatedBy(), c.GetCreatedAt(), c.GetUpdatedAt()), nil
		}
	}

	return nil, nil
}

func (r *ChatRepository) Save(c chat.Chat) (chat.Chat, error) {
	newChat := chat.NewChat(r.getLastId()+1, c.GetName(), c.GetType(), c.GetImage(), c.GetLastMessage(), c.GetCreatedBy(), c.GetCreatedAt(), c.GetUpdatedAt())

	r.chats = lo.Filter(r.chats, func(item chat.Chat, _ int) bool {
		return item.GetId() != newChat.GetId()
	})

	r.chats = append(r.chats, newChat)
	return c, nil
}

func (r *ChatRepository) Delete(id uint64) error {
	r.chats = lo.Filter(r.chats, func(item chat.Chat, _ int) bool {
		return item.GetId() != id
	})
	return nil
}

func NewChatRepository() *ChatRepository {
	chats := make([]chat.Chat, 0)
	chats = append(chats, chat.NewChat(1, "Chat 1", chat.Direct, "", nil, 1, time.Now(), time.Now()))

	return &ChatRepository{
		chats: chats,
	}
}
