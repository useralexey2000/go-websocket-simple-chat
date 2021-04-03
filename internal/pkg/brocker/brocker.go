package brocker

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-redis/redis"
)

// Message .
type Message struct {
	Ch   string
	Data string
}

// Brocker .
type Brocker interface {
	Pub(context.Context, string, string) error
	Sub(context.Context, string) (<-chan Message, error)
	UnSub(string)
}

// RedisBrocker .
type RedisBrocker struct {
	*redis.Client
	// TODO check if it is possible to unsubscribe without
	// explicit mapping pubsub in here
	channels map[string]*redis.PubSub
}

// NewRedisBrocker .
func NewRedisBrocker(client *redis.Client) *RedisBrocker {
	return &RedisBrocker{
		client,
	}
}

// Pub .
func (b *RedisBrocker) Pub(ctx context.Context, ch string, msg string) error {
	return b.Publish(ctx, ch, msg).Err()
}

// Sub .
func (b *RedisBrocker) Sub(ctx context.Context, ch string) (<-chan Message, error) {
	ps := b.Subscribe(ctx, ch)
	_, err := ps.Receive(ctx)
	if err != nil {
		ps.Close()
		return nil, err
	}
	channel := make(chan Message)
	b.channels[ch] = ps
	go func() {
		ch := ps.Channel()
		for msg := range ch {
			var m Message
			err := json.Unmarshal([]byte(msg.Payload), &m)
			if err != nil {
				fmt.Println(err)
				m = Message{}
			}
				channel <- m
			}
		}

	}()
	return channel, nil

}

func (b *RedisBrocker)UnSub(ch string) {
	b.channels[ch].Close()
}
