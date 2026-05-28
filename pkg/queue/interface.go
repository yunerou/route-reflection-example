package queue

import (
	"context"
	"log/slog"
	"reflect"
	"time"
)

type NetworkSerializable interface {
	Encode() []byte
	Decode([]byte) error
}

type QueueProvider[ChannelT ~string, MsgT NetworkSerializable] interface {
	Publish(ctx context.Context, channel ChannelT, message MsgT) error
	FetchOne(ctx context.Context, channel ChannelT) (message MsgT, err error)
	FetchMany(ctx context.Context, channel ChannelT, maxItem int) (message []MsgT, err error)
	BlockFetchOne(ctx context.Context, channel ChannelT, timeout time.Duration) (message MsgT, err error)

	Healthcheck(ctx context.Context) error

	Close()
}

type queueProvider[ChannelT ~string, MsgT NetworkSerializable] struct {
	adapter Adapter
}

func New[ChannelT ~string, MsgT NetworkSerializable](adapter Adapter) QueueProvider[ChannelT, MsgT] {
	ins := &queueProvider[ChannelT, MsgT]{
		adapter,
	}
	return ins
}

func (r *queueProvider[ChannelT, MsgT]) Publish(ctx context.Context, channel ChannelT, message MsgT) error {
	return r.adapter.Publish(ctx, string(channel), message.Encode())
}

func (r *queueProvider[ChannelT, MsgT]) FetchOne(ctx context.Context, channel ChannelT) (message MsgT, err error) {
	var m []byte
	m, err = r.adapter.FetchOne(ctx, string(channel))
	if err != nil {
		slog.Error("Queue FetchOne Adapter.FetchOne", slog.Any("err", err))
		return message, err
	}

	message, err = decodeNetworkSerializable[MsgT](m)
	if err != nil {
		slog.Error("Queue FetchOne NetworkSerializable.Decode", slog.Any("err", err))
		return message, err
	}
	// message can be nil if the channel is empty, in that case we return (nil,nil)
	return message, nil
}

func (r *queueProvider[ChannelT, MsgT]) BlockFetchOne(ctx context.Context, channel ChannelT, timeout time.Duration) (message MsgT, err error) {
	var m []byte
	m, err = r.adapter.BlockFetchOne(ctx, string(channel), timeout)
	if err != nil {
		slog.Error("Queue FetchOne Adapter.FetchOne", slog.Any("err", err))
		return message, err
	}

	message, err = decodeNetworkSerializable[MsgT](m)
	if err != nil {
		slog.Error("Queue FetchOne NetworkSerializable.Decode", slog.Any("err", err))
		return message, err
	}
	// message can be nil if the channel is empty, in that case we return (nil,nil)
	return message, nil
}

func (r *queueProvider[ChannelT, MsgT]) FetchMany(ctx context.Context, channel ChannelT, maxItem int) (message []MsgT, err error) {
	var data [][]byte
	data, err = r.adapter.FetchMany(ctx, string(channel), maxItem)
	if err != nil {
		slog.Error("Queue FetchMany Adapter.FetchMany", slog.Any("err", err))
		return nil, err
	}

	message = make([]MsgT, 0, len(data))
	for _, d := range data {
		var msg MsgT
		msg, err = decodeNetworkSerializable[MsgT](d)
		if err != nil {
			slog.Error("Queue FetchMany NetworkSerializable.Decode", slog.Any("err", err))
			return message, err
		}
		message = append(message, msg)
	}

	return message, nil
}

func (r *queueProvider[ChannelT, MsgT]) Close() {
	r.adapter.Close()
}

func (r *queueProvider[ChannelT, MsgT]) Healthcheck(ctx context.Context) error {
	return r.adapter.Healthcheck(ctx)
}

type HandlerFn[T NetworkSerializable] = func(hdlMsg T)

func decodeNetworkSerializable[T NetworkSerializable](data []byte) (T, error) {
	var (
		msg   T
		zeroT T
	)
	if reflect.TypeOf(msg).Kind() == reflect.Ptr {
		msg = reflect.New(reflect.TypeOf(msg).Elem()).Interface().(T)
	} else {
		msg = reflect.Zero(reflect.TypeOf(msg)).Interface().(T)
	}
	err := msg.Decode(data)
	if err != nil {
		return zeroT, err
	}
	return msg, nil
}
