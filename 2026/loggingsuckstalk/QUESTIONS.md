# Questions

Questions asked by the audience in the Q/A session.

- Is there any standard on how to implement loggingsucks?
- Not really, this concept is to show an alternative to the classic logging we all have been using in our backend apps. 

- It looks good, but what happens if the application crashes during a request, you would lose the log entry.
- It always depends on how you implement this concept, you could `recover` from panic, for example. But, normally yes. 

- Do you know how many fields the system can handle?
- I haven't used this approach in a backend that needs to log many fields, but I know that some companies are using this approach, so I guess it can handle big loads.
