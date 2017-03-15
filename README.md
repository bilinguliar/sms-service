# sms-service

[![Go Report Card](https://goreportcard.com/badge/github.com/bilinguliar/sms-service)](https://goreportcard.com/report/github.com/bilinguliar/sms-service)

HTTP server that exposes endpoint for SMS sending. 

* Accepts `POST` requests on URL: `/messages`
* Sends messages with rate limited to 1 SMS per second.
* Internally uses [MessageBird.com SMS Gateway](https://www.messagebird.com/).

Example request:

Method: `POST`

URL: `/messages`
```
{
    "originator": "YourService",
    "recipient": 334223445566,
    "message": "Dear customer, this text is really important."
}
```

Long messages will be split to so-called concatenated SMS. Anyway limit is 9 concatenated SMS: 1377 chars.
Note that chars from extended set will be automatically escaped so thay counted as 2 chars.
