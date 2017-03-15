package main

import (
	"errors"
	"reflect"
	"strconv"
	"sync"
	"testing"
	"time"

	mb "github.com/messagebird/go-rest-api"
)

func TestSplitToMessagesPanicsIfBodyIsToLong(t *testing.T) {
	defer func() {
		err := recover()
		if err == nil {
			t.Error("Expected panic never occurred")
		}
	}()

	body := `
||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||
||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||
||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||
||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||
||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||
||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||
||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||
||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||
||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||` // > 9 SMS

	splitToMsgs("", "", body)
}

func TestSplitToMessagesCreatesMsgRequestsWithProperData(t *testing.T) {
	body := `
1||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||
2||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||
3|||||||||||||||||||||||||||||||||||END
`

	expectedBodies := chunkBody(body, PlainConcatSMSLen)
	originator := "Mr. Smith"
	recipient := "Mr. Davis"

	msgs := splitToMsgs(originator, recipient, body)

	for i, m := range msgs {
		if m.Originator != originator {
			t.Errorf("Actual originator: %q, expected: %q", m.Originator, originator)
		}
		if m.Recipient != recipient {
			t.Errorf("Actual recipient: %q, expected: %q", m.Recipient, recipient)
		}
		if m.Body != expectedBodies[i] {
			t.Errorf("Actual body: %q, expected: %q", m.Body, expectedBodies[i])
		}
		expectedHeader := NewUDH(byte(len(expectedBodies)), byte(i+1))
		if m.Header != expectedHeader {
			t.Errorf("Actual header: %q, expected header: %q", m.Header.ToHexStr(), expectedHeader.ToHexStr())
		}
	}

}

// TestGetSymbCount checks if we take into account that some symbols must be escaped in SMS.
// That's why the are counted as 2 symbols.
func TestGetSymbCount(t *testing.T) {
	cases := []struct {
		Body     string
		Expected int
	}{
		{
			Body:     "1234567890",
			Expected: 10,
		},
		{
			Body:     "\n\\^[]{}|~€", // Each symbol in this string belongs to char set extension and counts as 2.
			Expected: 20,
		},
		{
			Body:     "",
			Expected: 0,
		},
		{
			Body:     "Here comes some text.]",
			Expected: 23,
		},
		{
			Body:     "Here comes some text.",
			Expected: 21,
		},
	}

	for _, c := range cases {
		actual := getBodyCount(c.Body)
		if actual != c.Expected {
			t.Errorf("Count is %d, expected: %d", actual, c.Expected)
		}
	}
}

func TestChunkBody(t *testing.T) {
	cases := []struct {
		Body   string
		Length int
		Result []string
	}{
		{
			Body:   "1234567890", // symbols that do not require escape char
			Length: 4,
			Result: []string{"1234", "5678", "90"},
		},
		{
			Body:   "1234[]1234", // 2 escaped symbols in the middle
			Length: 4,
			Result: []string{"1234", "[]", "1234"},
		},
		{
			Body:   "1234[]1234",
			Length: 5,
			Result: []string{"1234", "[]1", "234"},
		},
		{
			Body:   "1",
			Length: 4,
			Result: []string{"1"},
		},
		{
			Body:   "Hola|{}", // if next symbol in a chunk overflows limit - move to next chunk
			Length: 4,
			Result: []string{"Hola", "|{", "}"},
		},
	}

	for _, c := range cases {
		actual := chunkBody(c.Body, c.Length)
		if !reflect.DeepEqual(actual, c.Result) {
			t.Errorf("Got chunks: %q, expected: %q", actual, c.Result)
		}
	}
}

func TestSendText(t *testing.T) {
	cases := []struct {
		originator  string
		recipient   string
		body        string
		udhExpected bool
	}{
		{
			originator: "Karl",
			recipient:  "380730220022",
			body: `This text must be long enough to trigger msg concatenation logic. 
Why I am writing this, why not simpli copy it from Wikipedia. Silence was the answer.
Would be nice to add fe symbols from extended set. ~ yes, [GREAT], {curly are the best}.`,
			udhExpected: true,
		},
		{
			originator:  "LaconicGuy",
			recipient:   "380730220033",
			body:        `This text intentionally left short.`,
			udhExpected: false,
		},
	}

	for _, c := range cases {
		numOfSMS := len(chunkBody(c.body, PlainConcatSMSLen))

		var wg sync.WaitGroup
		wg.Add(numOfSMS)

		mock := &MockedMBClient{
			NewMessageFunc: func(originator string, recipients []string, body string, msgParams *mb.MessageParams) mb.Message {
				wg.Done()
				rcpntNum, err := strconv.Atoi(recipients[0])
				if err != nil {
					t.Error("Expected recipient string to be convertable to int")
				}

				return mb.Message{
					Originator:  originator,
					Recipients:  mb.Recipients{Items: []mb.Recipient{{Recipient: rcpntNum}}},
					Body:        body,
					TypeDetails: msgParams.TypeDetails,
				}
			},
		}

		client := NewMsgBirdClient(mock, 100, 1*time.Microsecond)

		client.SendText(c.originator, c.recipient, c.body)

		// Wait for all messages to be processed.
		wg.Wait()

		for i, m := range mock.SentMsgs {
			if m.Originator != c.originator {
				t.Errorf("Got originator: %q, expected: %q", m.Originator, c.originator)
			}
			actualRecpnt := strconv.Itoa(m.Recipients.Items[0].Recipient)
			if actualRecpnt != c.recipient {
				t.Errorf("Got recipient: %q, expected: %q", m.Originator, c.originator)
			}

			if c.udhExpected {
				actualUDH := m.TypeDetails["udh"].(string)

				if actualUDH != NewUDH(byte(numOfSMS), byte(i+1)).ToHexStr() {
					t.Error("Wrong UDH header")
				}
			} else {
				actualUDH := m.TypeDetails["udh"]
				if actualUDH != nil {
					t.Error("UDH was set even it is not needed.")
				}
			}
		}
	}
}

func TestMBClientReturnedError(t *testing.T) {
	mock := &MockedErrorproneMBClient{}
	client := NewMsgBirdClient(mock, 100, 1*time.Microsecond)

	// We do not handle errors form API for now, so just making sure it will not panic. Not other checks.
	client.SendText("NotImprtnt", "380660000000", "Hola!")
}

type MockedMBClient struct {
	SentMsgs       []mb.Message
	NewMessageFunc func(originator string, recipients []string, body string, msgParams *mb.MessageParams) mb.Message
}

func (mc *MockedMBClient) NewMessage(originator string, recipients []string, body string, msgParams *mb.MessageParams) (*mb.Message, error) {
	m := mc.NewMessageFunc(originator, recipients, body, msgParams)
	mc.SentMsgs = append(mc.SentMsgs, m)

	return &m, nil
}

type MockedErrorproneMBClient struct{}

func (mec *MockedErrorproneMBClient) NewMessage(originator string, recipients []string, body string, msgParams *mb.MessageParams) (*mb.Message, error) {
	return &mb.Message{Errors: []mb.Error{{Code: 42, Description: "Expected error", Parameter: ""}}}, errors.New("something went wrong")
}
