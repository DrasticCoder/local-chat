# Local Network Chat

## Overview

The **Local Network Chat** project enables communication between devices on the same local network via WebSockets. The server handles client connections and broadcasts messages to all connected clients or to specific clients by their device names. It uses an SQLite database to store message history for persistence.

This project consists of a **Go server** that manages the WebSocket connections and an **HTML frontend** for sending and receiving messages.

## Features

- WebSocket-based communication for real-time chat.
- Supports sending messages to all devices (`ALL`) or to specific devices by their name.
- Each device is assigned a random nickname on connection.
- Device list broadcasted to all connected clients for easy selection.
- Messages are stored in an SQLite database.

## Technologies Used

- **Backend**: Go (Golang)
- **WebSocket**: Gorilla WebSocket
- **Database**: SQLite
- **Frontend**: HTML, JavaScript (Vanilla JS)


## Installation Guide

### Prerequisites

Before setting up this project, make sure you have the following installed:

- **Go** (Golang) - [Download Go](https://go.dev/dl/)
- **SQLite** (SQLite3) - [Download SQLite](https://www.sqlite.org/download.html)
- **Go Modules** (Optional for dependency management)

### Steps to Set Up

0. **Clone the repository**:

1. **Install dependencies**:

To install the necessary dependencies, run:

```bash
go mod tidy
```

2. **Run the Go server**:


```bash
go run main.go
```
The server will start and listen on http://localhost:8080.