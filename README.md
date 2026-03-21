# go-gmail 📧

[![Go Reference](https://pkg.go.dev/badge/github.com/gowriprasathdev/go-gmail.svg)](https://pkg.go.dev/github.com/gowriprasathdev/go-gmail)
[![Go Report Card](https://goreportcard.com/badge/github.com/gowriprasathdev/go-gmail)](https://goreportcard.com/report/github.com/gowriprasathdev/go-gmail)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)

A robust, fluent, and rate-limited Go wrapper for the official Google Gmail API (`google.golang.org/api/gmail/v1`). 

Standard Go email packages force you to manually construct MIME boundaries and handle raw Base64 encoding. `go-gmail` abstracts the complexity away, providing a clean, chainable builder pattern while actively protecting your Google account from rate-limit bans.

## ✨ Features

* **Fluent Builder API:** Construct complex emails (HTML, CC, Attachments) in a single readable chain.
* **Account Protection:** Built-in rate limiting and graceful degradation to prevent your account from being flagged by Google for spamming or hitting API quotas.
* **Smart Attachments:** Seamlessly attach files from a local path or directly from an `io.Reader` (perfect for cloud storage microservices).
* **Context-Aware:** Full `context.Context` support for modern, timeout-safe network calls.
* **Zero MIME Headaches:** Automatically handles RFC 2822 formatting and Base64URL encoding under the hood.

## 📦 Installation

```bash
go get [github.com/gowriprasathdev/go-gmail](https://github.com/gowriprasathdev/go-gmail)
