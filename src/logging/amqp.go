package logging

import (
	"errors"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
)

type AMQPHook struct {
	levels []logrus.Level

	entries chan *logrus.Entry
}

const amqpUrl = "amqp://localhost/"

func NewAMQPHook() *AMQPHook {
	hook := AMQPHook{}

	hook.levels = []logrus.Level{logrus.InfoLevel}

	// buffer 20 log entries before dropping them
	hook.entries = make(chan *logrus.Entry, 20)

	go func() {

		var connection *amqp.Connection
		var channel *amqp.Channel

		var backOffStrategy = backoff.NewExponentialBackOff()
		backOffStrategy.MaxElapsedTime = 0
		backOffStrategy.MaxInterval = 2 * time.Minute
		for true {
			backOffStrategy.Reset()
			backoff.Retry(func() error {
				conn, err := amqp.Dial(amqpUrl)
				if err != nil {
					return err
				}
				ch, err := conn.Channel()

				connection = conn
				channel = ch
				return err
			}, backOffStrategy)

			err := publish(channel, hook.entries)
			// if entries channel was closed for some reason
			if err == nil {
				return
			}

		}

	}()

	return &hook
}

func publish(channel *amqp.Channel, entries chan *logrus.Entry) error {
	for entry := range entries {

		encoded, encodeErr := formatter.Format(entry)
		if encodeErr != nil {
			continue
		}

		msg := amqp.Publishing{
			Timestamp:   entry.Time.UTC(),
			ContentType: "application/json",
			Body:        encoded,
		}

		err := channel.Publish("driver", "", false, false, msg)
		if err != nil {
			return err
		}

	}
	return nil
}

func (hook *AMQPHook) Levels() []logrus.Level {
	return hook.levels
}

func (hook *AMQPHook) Fire(entry *logrus.Entry) error {
	select {
	case hook.entries <- entry:
		return nil
	default:
		return errors.New("Outgoing AMQP channel full, dropping entry.")
	}
}
