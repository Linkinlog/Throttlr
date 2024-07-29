## What should it do? (MVP)
1. A user should be able to register an endpoint with us. This includes the endpoint url and the rate at which is is allowed to be hit and the optional data.
2. Once an endpoint is registered, they will get a unique url that they can replace in their code. 
3. This url will require an API key for book keeping.
4. Within the rate, we keep taking from our bucket each time our endpoint is hit, once our bucket is out, we will respond with a standard http response.

## Usage of an API Key
- Can be whatever is the most safe UUID since it should be copy/pasted somewhere safe
- associated with a user
- generate with googles package
- each user only has one
- should be made on user creation with ability to invalidate
## Register
- Should be a post request with the API key
- own entity
- `/register?key=asdf` `data={"endpoint": "https://google.com", "interval": "daily", "limit": 100}`
## Generation of unique url
- associated with endpoint
## Token bucket algorithm
- own entity
