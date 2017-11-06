package smsd

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"
)

// MaxMsgLen maximum body that can be sent in 9 concatenated messages.
const MaxMsgLen = PlainConcatSMSLen * 9

var (
	// MSISDNRegex validates string representation of MSISDN.
	MSISDNRegex = regexp.MustCompile(`^[1-9]{1}[0-9]{3,14}$`)
	// OriginatorRegex validates alphanumeric originator string.
	OriginatorRegex = regexp.MustCompile(`^[a-zA-Z0-9]{1,11}$`)
)

// Handler is responsible for processing incommint HTTP message requests.
type Handler struct {
	messenger Messenger
}

// Messenger declares single method that we utilize to request SMS message submission.
type Messenger interface {
	SendText(originator, recipient, body string)
}

// NewHandler constructs Handler instance with provided Messenger implementation.
func NewHandler(m Messenger) *Handler {
	return &Handler{
		messenger: m,
	}
}

// MsgRequest HTTP request body that we accept on endpoint.
type MsgRequest struct {
	Originator string
	Message    string
	Recipient  int
}

// Validate checks if request values are in conflict with our specification.
// Return only static errors here, as error text will be returned to client and we do not want XSS.
func (r MsgRequest) Validate() error {
	var err error

	if r.Originator == "" {
		err = errors.New("originator can not be blank")
	}

	if !MSISDNRegex.MatchString(r.Originator) && !OriginatorRegex.MatchString(r.Originator) {
		err = errors.New("originator is not an MSISDN or it is to long")
	}

	if r.Message == "" {
		err = errors.New("message can not be blank")
	}

	if getBodyCount(r.Message) > MaxMsgLen {
		err = fmt.Errorf("message is longer than %d", MaxMsgLen)
	}

	msisdn := strconv.Itoa(r.Recipient)
	if !MSISDNRegex.MatchString(msisdn) {
		err = errors.New("recipient MSISDN is wrong")
	}

	return err
}

// HandleMsg accepts POST requests with serialized message details.
func (h *Handler) HandleMsg(w http.ResponseWriter, req *http.Request) {
	if req.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var msg MsgRequest
	err := json.NewDecoder(req.Body).Decode(&msg)
	if err != nil {
		log.Println("Request body is not valid, error:", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if err := msg.Validate(); err != nil {
		log.Println("Request values are not valid, error:", err)

		// Returning error text to client.
		// Avoid dynamic error values here as this will lead to XSS vulnerability.
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Would be great to return ID for status check. But we will need some kind of persistence to achieve this.
	// Omitting this for now.
	//
	// We are keeping client connections open for some time if queue is full.
	// Will be good to pass context with timeout in order to return correct HTTP code: 429 Too Many Requests.
	// But need to make sure that we send all or nothing in case of concatenated messages.
	h.messenger.SendText(msg.Originator, strconv.Itoa(msg.Recipient), msg.Message)
}
