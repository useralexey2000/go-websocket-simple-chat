package brocker

import (
	"context"
	"fmt"

	"github.com/go-redis/redis/v8"
)

// Brocker .
type Brocker interface {
	Pub(context.Context, string, interface{}) error
	Sub(context.Context, string) (<-chan string, error)
	UnSub(string)
}

// RedisBrocker .
type RedisBrocker struct {
	client *redis.Client
	// TODO check if it is possible to unsubscribe without
	// explicit mapping pubsub in here
	channels map[string]*redis.PubSub
}

// NewRedisBrocker .
func NewRedisBrocker(client *redis.Client) Brocker {
	return &RedisBrocker{
		client:   client,
		channels: make(map[string]*redis.PubSub),
	}
}

// Pub .
func (b *RedisBrocker) Pub(ctx context.Context, ch string, msg interface{}) error {
	return b.client.Publish(ctx, ch, msg).Err()
}

// Sub .
func (b *RedisBrocker) Sub(ctx context.Context, ch string) (<-chan string, error) {
	ps := b.client.Subscribe(ctx, ch)
	_, err := ps.Receive(ctx)
	if err != nil {
		fmt.Println(err)
		ps.Close()
		return nil, err
	}
	channel := make(chan string)
	b.channels[ch] = ps
	go func() {
		ch := ps.Channel()
		for msg := range ch {
			channel <- msg.Payload
		}

	}()
	return channel, nil

}

func (b *RedisBrocker) UnSub(ch string) {
	b.channels[ch].Close()
}
