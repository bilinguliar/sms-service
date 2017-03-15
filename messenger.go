package main

import (
	"log"
	"strings"
	"time"

	mb "github.com/messagebird/go-rest-api"
)

const (
	// MaxSeqSMSCount max supported concatenated SMS
	MaxSeqSMSCount = 9
	// PlainSMSLen number of chars in 7-bit encoding for single SMS without concat.
	PlainSMSLen = 160
	// PlainConcatSMSLen number of chars in 7-bit encoding for single SMS with concat.
	PlainConcatSMSLen = 153

	udhField     = "udh"
	escapedChars = "\n\\^~[]{}|~€"
)

// MBClient declares MessageBird NewMessage method.
type MBClient interface {
	NewMessage(originator string, recipients []string, body string, msgParams *mb.MessageParams) (*mb.Message, error)
}

// Client holds actual MessageBird.com client and channel that we use for rate limiting.
type Client struct {
	mbClient MBClient
	msgChan  chan Msg
	sendRate time.Duration
}

// NewMsgBirdClient creates new rate limited instance of messaging client that uses MessageBird.com API.
func NewMsgBirdClient(mbClient MBClient, queueSize int, sendRate time.Duration) *Client {
	c := &Client{
		mbClient: mbClient,
		msgChan:  make(chan Msg, queueSize),
	}

	go c.startWorker(sendRate)

	return c
}

// process prepares data and sends generated SMS from sender to a recipient with provided text.
// Not exported as needs to be used through rate limiter.
func (c *Client) process(mr Msg) {
	msgParams := &mb.MessageParams{}

	if mr.Header.IsSet() {
		details := make(map[string]interface{})
		details[udhField] = mr.Header.ToHexStr()
		msgParams.TypeDetails = details
	}

	m, err := c.mbClient.NewMessage(
		mr.Originator,
		[]string{mr.Recipient},
		mr.Body,
		msgParams,
	)
	if err != nil {
		// Possibly resubmit request to the queue with some delay,
		// keeping track of number of attempts.
		// This will require extending Msg to have attempts counter.
		// For now simply logging errors.

		if err != mb.ErrResponse {
			log.Println("Failed to send SMS. Unrecoverable error:", err)
			return
		}
		for _, mbError := range m.Errors {
			log.Printf("Error: %#v\n", mbError)
		}
	}
}

// SendText submits SMS request. It will be send sometime in the future.
// Call to this function may be long if channel is full. Consider passing context with timeout.
func (c *Client) SendText(originator, recipient, body string) {
	if getBodyCount(body) <= PlainSMSLen {
		c.msgChan <- Msg{
			Originator: originator,
			Recipient:  recipient,
			Body:       body,
		}
	} else {
		msgs := splitToMsgs(originator, recipient, body)

		for _, m := range msgs {
			c.msgChan <- m
		}
	}
}

// startWorker runs a loop that sends short messages with constant rate.
func (c *Client) startWorker(sendRate time.Duration) {
	t := time.Tick(sendRate)

	for {
		<-t // Tick
		c.process(<-c.msgChan)
	}
}

// getBodyCount evaluates each symbol in provided body in terms of GSM_03.38 encoding.
func getBodyCount(body string) int {
	var sum int

	for _, r := range body {
		sum += getSymbolSize(r)
	}

	return sum
}

// getSymbolSize returns number of characters needed to represent given symbol in GSM_03.38
// Symbols from extended set require preceding escape symbol.
func getSymbolSize(r rune) int {
	if strings.ContainsRune(escapedChars, r) {
		return 2
	}
	return 1
}

// splitToMsgs creates multiple messages of valid size based on body size
func splitToMsgs(originator, recipient, body string) []Msg {
	bodyParts := chunkBody(body, PlainConcatSMSLen)
	msgCount := len(bodyParts)

	if msgCount > MaxSeqSMSCount {
		// Panicking is bad but in this case better to fail than violite SMS Gateway API.
		panic("message body does not fit in 9 messages")
	}

	msgs := make([]Msg, 0, msgCount)

	for i, b := range bodyParts {
		msg := Msg{
			Header:     NewUDH(byte(msgCount), byte(i+1)), // Casting is safe here as we panic if count is over 9.
			Originator: originator,
			Recipient:  recipient,
			Body:       b,
		}
		msgs = append(msgs, msg)
	}

	return msgs
}

// chunkBody splits string to chunks of requested size
// string may be less than requested if next symbol overflows limit.
func chunkBody(b string, maxChars int) []string {
	var (
		chunks       []string
		chars        int
		nextSymbSize int
	)

	chunk := make([]rune, 0, maxChars)
	for i, r := range b {
		nextSymbSize = chars + getSymbolSize(rune(b[i]))

		if chars == maxChars || nextSymbSize > maxChars {
			// this chunk is full
			chunks = append(chunks, string(chunk))
			chunk = make([]rune, 0, maxChars)
			chars = 0
		}

		chars += getSymbolSize(r)
		chunk = append(chunk, r)
	}

	// last chunk
	if len(chunk) != 0 {
		chunks = append(chunks, string(chunk))
	}

	return chunks
}
