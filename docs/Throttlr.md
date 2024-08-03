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

# Diagrams
## API Key Generation
![Sequence diagram (3)](https://github.com/user-attachments/assets/a50c4412-2313-4309-a432-d8e5582be93a)

## Endpoint Registration
![Sequence diagram (4)](https://github.com/user-attachments/assets/58974aa0-f15f-4e33-b136-ee8c67d3bf65)

## Endpoint Throttling
![Sequence diagram (5)](https://github.com/user-attachments/assets/b393595a-fd6e-4ab5-bd66-109fae137aa6)
