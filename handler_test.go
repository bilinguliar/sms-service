package main

import "testing"

func TestMsgRequestValidate(t *testing.T) {
	cases := []struct {
		req   MsgRequest
		valid bool
	}{
		{
			req: MsgRequest{
				Originator: "MessageBird",   // 11 symbols
				Recipient:  380660000000,    // valid MSISDN
				Message:    "get shit done", // valid in all cases
			},
			valid: true,
		},
		{
			req: MsgRequest{
				Originator: "MessageBird2",  // 12 symbols
				Recipient:  380660000000,    // valid MSISDN
				Message:    "get shit done", // valid in all cases
			},
			valid: false,
		},

		{
			req: MsgRequest{
				Originator: "380660000001",  // valid MSISDN
				Recipient:  380660000000,    // valid MSISDN
				Message:    "get shit done", // valid
			},
			valid: true,
		},
		{
			req: MsgRequest{
				Originator: "",
				Recipient:  380660000000,
				Message:    "Valid message.",
			},
			valid: false,
		},
		{
			req: MsgRequest{
				Originator: "Valid",
				Recipient:  380660000000,
				Message: "||||||||||||||||||||||||||||||||||||||||||||||" +
					"|||||||||||||||||||||||||||||||||||||||||||||||" +
					"|||||||||||||||||||||||||||||||||||||||||||||||" +
					"|||||||||||||||||||||||||||||||||||||||||||||||" +
					"|||||||||||||||||||||||||||||||||||||||||||||||" +
					"|||||||||||||||||||||||||||||||||||||||||||||||" +
					"|||||||||||||||||||||||||||||||||||||||||||||||" +
					"|||||||||||||||||||||||||||||||||||||||||||||||" +
					"|||||||||||||||| I AM TOO LONG ||||||||||||||||" +
					"|||||||||||||||||||||||||||||||||||||||||||||||" +
					"|||||||||||||||||||||||||||||||||||||||||||||||" +
					"|||||||||||||||||||||||||||||||||||||||||||||||" +
					"|||||||||||||||||||||||||||||||||||||||||||||||" +
					"|||||||||||||||||||||||||||||||||||||||||||||||" +
					"|||||||||||||||||||||||||||||||||||||||||||||||" +
					"|||||||||||||||||||||||||||||||||||||||||||||||" +
					"|||||||||||||||||||||||||||||||||||||||||||||||",
			},
			valid: false,
		},

		{
			req: MsgRequest{
				Originator: "Valid",
				Recipient:  380660000000,
				Message:    "",
			},
			valid: false,
		},
		{
			req: MsgRequest{
				Originator: "Valid",
				Recipient:  380, // too short
				Message:    "Valid message.",
			},
			valid: false,
		},
		{
			req: MsgRequest{
				Originator: "OriginatorToLong",
				Recipient:  380660000000,
				Message:    "Valid message.",
			},
			valid: false,
		},
	}

	for i, c := range cases {
		err := c.req.Validate()

		if c.valid && err != nil {
			t.Errorf("Validation failed, expected it to pass. Case: %d, error: %q", i, err)
		}

		if !c.valid && err == nil {
			t.Error("Validation passed, expected it to fail. Case: ", i)
		}
	}
}

func TestMSISDNRegex(t *testing.T) {
	validCases := []string{
		"1234",
		"123456789012345",
		"380662556677",
	}

	for _, c := range validCases {
		if !MSISDNRegex.MatchString(c) {
			t.Errorf("Valid MSISDN: %q did not pass validation.", c)
		}
	}

	invalidCases := []string{
		"1",
		"12",
		"123",              // to short
		"0123456789",       // starts with zero
		"1234567890123456", // over 15 symbols
	}

	for _, c := range invalidCases {
		if MSISDNRegex.MatchString(c) {
			t.Errorf("Invalid MSISDN: %q passed validation.", c)
		}
	}
}
