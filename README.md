# sms-service

HTTP server that exposes endpoint for SMS sending. 

* Accepts POST requests on URL: /messages
* Sends messages with rate limited to 1 SMS per second.
* Internally uses MessageBird.com SMS Gateway.
