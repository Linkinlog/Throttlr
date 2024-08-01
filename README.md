[![GoDoc](https://godoc.org/github.com/linkinlog/throttlr?status.svg)](https://godoc.org/github.com/linkinlog/throttlr) [![Go Report Card](https://goreportcard.com/badge/github.com/linkinlog/throttlr)](https://goreportcard.com/report/github.com/linkinlog/throttlr)

# Table of Contents
1. [What is Throttlr?](#What-is-Throttlr)
2. [Goals](#Goals)
3. [Resources Used](#resources-used)

# What is Throttlr?

Have you ever faced costly limitations with a 3rd party API? Struggling to keep your distributed system within free tier limits?

## Meet Throttlr.

Throttlr is a rate limiter that helps you control requests per second, minute, hour, day, or month. It's simple to use, can be self-hosted, or accessed at https://throttlr.dahlton.org, and easily integrates into your existing systems as a drop-in replacement for your endpoints.

## How it works
Just register an endpoint with a limit, and Throttlr takes care of the rest. When the limit is reached, it returns a 429 status code, passing through all HTTP headers, data, and query parameters to your endpoint. All you need to do is replace the URL your service is using with the URL Throttlr provides.

## Stay within limits effortlessly—try Throttlr today!

# Goals
- [ ] [Users will get API keys on account creation](https://github.com/Linkinlog/Throttlr/milestone/1)
- [ ] [Users can register endpoints, receiving a throttled endpoint to replace their existing ones](https://github.com/Linkinlog/Throttlr/milestone/2)
- [ ] [Users can set rate limits for a given unit of time, when said limit is hit, we return HTTP 429 (Too many requests)](https://github.com/Linkinlog/Throttlr/milestone/3)
- [ ] [Users can self host *or* use our servers online
](https://github.com/Linkinlog/Throttlr/milestone/4)
<details>
  <summary><h2>Resources used</h2></summary>
  <details>
    <summary><h3>General help</h3></summary>
    <a href="https://www.oreilly.com/library/view/cloud-native-go/9781492076322/">Cloud Native Go</a>
  </details>
  <details>
    <summary><h3>Database</h3></summary>
    <a href="https://www.alexedwards.net/blog/organising-database-access">Organising database access in Go</a>
  </details>
  <details>
    <summary><h3>Auth / sessions</h3></summary>
    <a href="https://github.com/golangci/golangci-api/tree/master">The archived golangci-api</a>
    <a href="https://github.com/CurtisVermeeren/gorilla-sessions-tutorial/tree/master">gorilla-sessions-tutorial</a>
    <a href="https://github.com/svenrisse/bookshelf/tree/main">bookshelf</a>
  </details>
</details>
