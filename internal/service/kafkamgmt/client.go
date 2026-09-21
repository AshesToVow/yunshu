package kafkamgmt

import (
	"context"
	"fmt"
	"strings"
	"time"

	"yunshu/internal/model"

	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl"
	"github.com/segmentio/kafka-go/sasl/plain"
	"github.com/segmentio/kafka-go/sasl/scram"
)

type brokerEndpoint struct {
	brokers       []string
	username      string
	password      string
	saslMechanism string
	timeout       time.Duration
}

func (e brokerEndpoint) dialer() (*kafka.Dialer, error) {
	timeout := e.timeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	d := &kafka.Dialer{Timeout: timeout, DualStack: true}
	user := strings.TrimSpace(e.username)
	if user == "" {
		return d, nil
	}
	mech := strings.ToLower(strings.TrimSpace(e.saslMechanism))
	if mech == "" || mech == "none" {
		mech = "plain"
	}
	var mechanism sasl.Mechanism
	var err error
	switch mech {
	case "plain":
		mechanism = plain.Mechanism{Username: user, Password: e.password}
	case "scram-sha-256":
		mechanism, err = scram.Mechanism(scram.SHA256, user, e.password)
	case "scram-sha-512":
		mechanism, err = scram.Mechanism(scram.SHA512, user, e.password)
	default:
		mechanism = plain.Mechanism{Username: user, Password: e.password}
	}
	if err != nil {
		return nil, err
	}
	d.SASLMechanism = mechanism
	return d, nil
}

func (e brokerEndpoint) client() (*kafka.Client, error) {
	if len(e.brokers) == 0 {
		return nil, fmt.Errorf("kafka brokers empty")
	}
	dialer, err := e.dialer()
	if err != nil {
		return nil, err
	}
	timeout := e.timeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	t := &kafka.Transport{}
	if dialer.SASLMechanism != nil {
		t.SASL = dialer.SASLMechanism
	}
	return &kafka.Client{
		Addr:      kafka.TCP(e.brokers...),
		Timeout:   timeout,
		Transport: t,
	}, nil
}

func (e brokerEndpoint) dialLeader(ctx context.Context) (*kafka.Conn, error) {
	if len(e.brokers) == 0 {
		return nil, fmt.Errorf("kafka brokers empty")
	}
	dialer, err := e.dialer()
	if err != nil {
		return nil, err
	}
	return dialer.DialContext(ctx, "tcp", e.brokers[0])
}

func endpointFromRow(row *model.KafkamgmtConnection, password string) brokerEndpoint {
	timeout := time.Duration(row.TimeoutSec) * time.Second
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	return brokerEndpoint{
		brokers:       splitBrokers(row.Brokers),
		username:      row.Username,
		password:      password,
		saslMechanism: row.SASLMechanism,
		timeout:       timeout,
	}
}

func splitBrokers(s string) []string {
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == ',' || r == '\n' || r == ';'
	})
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if v := strings.TrimSpace(p); v != "" {
			out = append(out, v)
		}
	}
	return out
}

func joinBrokers(addrs []string) string {
	return strings.Join(addrs, ",")
}
