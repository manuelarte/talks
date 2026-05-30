# Logging Sucks Talk

Lighting talk done in the Golang Meetup to explain the concept of Wide Log Events described in:
[loggingsucks](https://loggingsucks.com/).

## Architecture

The app is a backend application that mocks a money transfer system. The steps are the following:

* An user requests a transfer from one account to another.
* The app sends the transfer to an external payment gateway.
* This payment system can fail, timeout, or complete successfully the transaction.
* If the response from the payment gateway is positive:
  * The app sends a Kafka event to notify that the transfer was successful.
  * The accounts' balance get updated.
* Every step can fail, from the payment gateway, to sending a kafka event or updating the accounts' balance.

## Logging

The idea is to show two ways of logging a request.

### Traditional Logging

Traditional logging implies that we log the steps in the code that are happening, things like:

* Initiating payment gateway request
* sending kafka event
* updating account balance
* etc

### Wide Event Logging

The wide event logging implies that, we don't log every step of the code, but a final business log event at
the end of the request. This event would contain all the information needed to completely understand the event.

If we take into account the steps explained in [Architecture](#architecture), the wide event would contain fields like:

- whether the payment gateway was successful or not.
- whether the kafka event was successful or not.
- whether the account update was successful or not.
- etc.

This allows us to have a better understanding of what happened in the request.

Another advantage is that, we could skip logging the "normal" scenarios, in which everything is correct
(or just logging a small percentage of it), but focus more on the "special" cases.

Since we have all the fields available in each log entry, we can also ask/filter business questions to our logs.

## Swagger

Swagger can be found http://localhost:8080/swagger/ and a mock client in [client](client.http).
There is also a small cli tool to create many request in parallel to compare the output of either using
normal logging or wide event logging.
