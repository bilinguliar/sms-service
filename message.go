package main

import "fmt"

// NewUDH constructs header that indicates that SMS is a part of sequence.
// Messages will be concatenated in correct order according to indexes.
//
// total - indicates how many messages are there in sequence. Max 9 for MessageBird.com.
// current - index of current message.
//
// Zero values are not allowed for total and current. Current can not be more than total. Max current is 9.
func NewUDH(total, current byte) UDH {
	if total == 0 || current == 0 || current > 9 || total > 9 || current > total {
		// this should never happen
		// but better fail if we got here.
		panic("header values violation")
	}
	return UDH{
		0x05,    // OveralLength
		0x00,    // Type
		0x03,    // HeaderLength
		total,   // TotalMsg
		current, // SequentialNum
	}
}

// UDH is a header that holds service information.
// In our case it is used to create concatenated messages so has constant length.
type UDH [5]byte

// ToHexStr returns string with header bytes in hex.
func (h UDH) ToHexStr() string {
	return fmt.Sprintf("%x", h)
}

// IsSet returns true if default values were modified.
func (h UDH) IsSet() bool {
	return h[3] != 0 && h[4] != 0
}

// Msg is a data transfer structure that holds data required by Messenger Client to send SMS.
//
// Depending on datacode, type and presence of UDH header message body can have different number of symbols.
// SMS is 1120 bit
//
// Unicode (16 bit) (70)
// Binary (8 bit) (140)
// GSM_03.38 (7 bit) (160)
//
// For now supporting only GSM_03.38
type Msg struct {
	Header     UDH
	Body       string
	Originator string
	Recipient  string
}
