# 🎵 Roulettify - Spotify Multiplayer Game

A real-time multiplayer web game where players guess which friend has listened to randomly selected tracks the most, based on their Spotify listening history.

## 🎮 How It Works

1. **Authenticate** with Spotify
2. **Join one of 3 persistent rooms** (max 10 players per room)
3. **Listen** to 30-second track previews
4. **Guess** which player has that track ranked highest in their top 50
5. **Earn points** for correct guesses (10 pts + 5 pts speed bonus)
6. **Compete** across 10 rounds to become the champion

## ✨ Features

- **Spotify OAuth2 Integration**: Securely authenticate and fetch your top 50 tracks
- **Real-time Multiplayer**: WebSocket-powered live game updates
- **3 Persistent Rooms**: Always-available game rooms (Room 1, Room 2, Room 3)
- **Capacity Management**: Max 10 players per room (30 concurrent players total)
- **Audio Previews**: 30-second Spotify track snippets for each round (scraped from embeds)
- **Dynamic Scoring**: 10 base points + 5 bonus for fastest correct guess
- **Persistent Rankings**: Tracks final standings and winner announcement
- **Optimized Architecture**: Persistent rooms with no cleanup overhead for efficient hosting
- **Smart Tie-Breaking**: Dense ranking system ensures fair placement for tied scores
- **Live Leaderboard**: Real-time sorting of players by score during the game

## 🛠 Technology Stack

### Backend
- **Go 1.25+** with Gin web framework
- **WebSocket** (coder/websocket) for real-time communication
- **Spotify API** (zmb3/spotify/v2) with OAuth2 authentication
- **Web Scraping** for preview URLs (colly framework)

### Frontend
- **React 18** with TypeScript
- **Vite** build tool with HMR
- **Tailwind CSS** for styling
- **WebSocket client** for game updates

### Infrastructure
- **Docker** multi-stage builds
- **Cloud-ready** (optimized for low-memory deployments)

## 📁 Project Structure

```
roulettify/
├── cmd/api/main.go                 # Entry point
├── internal/
│   ├── server/
│   │   ├── server.go              # Server initialization
│   │   └── routes.go              # HTTP/WebSocket routes
│   ├── auth/
│   │   ├── spotify.go             # Spotify OAuth & API
│   │   └── scraper.go             # Preview URL scraping
│   └── game/
│       ├── room.go                # Game room logic
│       ├── models.go              # Data structures
│       ├── manager.go             # 3 persistent rooms
│       └── *_test.go              # Test files
├── frontend/
│   ├── src/
│   │   ├── App.tsx                # Main component
│   │   ├── components/
│   │   │   ├── Lobby.tsx          # Room selection UI
│   │   │   └── GameRoom.tsx       # Game interface
│   │   └── main.tsx               # React entry point
│   └── package.json
├── Dockerfile                      # Multi-stage build
├── docker-compose.yml
├── Makefile
└── .env
```

## 🔌 API Endpoints

### REST

| Method | Endpoint | Purpose |
|--------|----------|---------|
| GET | `/` | Health check / SPA Entry |
| GET | `/health` | Detailed metrics (uptime, room stats) |
| GET | `/rooms` | List all 3 persistent rooms with player counts |
| GET | `/auth/spotify` | Start Spotify OAuth flow |
| GET | `/auth/callback` | Spotify OAuth callback handler |

### WebSocket (`/ws`)

**Client → Server**:

```json
{
  "type": "join_room",
  "payload": {
    "room_id": "Room 1",
    "player_id": "user123",
    "player_name": "John",
    "access_token": "spotify_token"
  }
}
```

```json
{
  "type": "ready",
  "payload": {
    "player_id": "user123",
    "is_ready": true
  }
}
```

```json
{
  "type": "start_game",
  "payload": {
    "room_id": "Room 1",
    "total_rounds": 10
  }
}
```

```json
{
  "type": "submit_guess",
  "payload": {
    "room_id": "Room 1",
    "player_id": "user123",
    "guessed_player_id": "friend456"
  }
}
```

**Server → Client**:

```json
{
  "type": "player_joined",
  "payload": {
    "players": [
      { "id": "user123", "name": "John", "score": 0, "is_ready": false, "is_leader": true }
    ],
    "player_count": 1
  }
}
```

```json
{
  "type": "player_ready",
  "payload": {
    "player_id": "user123",
    "is_ready": true
  }
}
```

```json
{
  "type": "game_started",
  "payload": {
    "total_rounds": 10,
    "players": [...]
  }
}
```

```json
{
  "type": "round_started",
  "payload": {
    "round": 1,
    "total_rounds": 10,
    "track": {
      "id": "123",
      "name": "Song Name",
      "artists": ["Artist 1"],
      "image_url": "...",
      "preview_url": "..."
    },
    "players": [...]
  }
}
```

```json
{
  "type": "guess_received",
  "payload": {
    "player_id": "user123",
    "guesses_count": 1,
    "total_players": 2
  }
}
```

```json
{
  "type": "round_complete",
  "payload": {
    "round": 1,
    "winner_id": "user123",
    "winner_rank": 5,
    "correct_guessers": ["user123"],
    "points_awarded": {"user123": 15},
    "updated_scores": {"user123": 15, "friend456": 0},
    "guess_durations": {"user123": 2.5}
  }
}
```

```json
{
  "type": "game_over",
  "payload": {
    "winner_id": "user123",
    "final_scores": {"user123": 85, "friend456": 60},
    "players": [...]
  }
}
```

```json
{
  "type": "game_reset",
  "payload": {
    "players": [...]
  }
}
```

```json
{
  "type": "error",
  "payload": {
    "message": "Room is full"
  }
}
```

## 🎯 Scoring System

| Event | Points |
|-------|--------|
| Correct guess | +10 |
| Speed bonus (fastest) | +5 |
| Wrong guess | 0 |

**Winner Determination**: The player whose top 50 contains the track with the **lowest rank number** (most listened to) wins the round.

## 🚀 Quick Start

### Local Development

```bash
# Clone the repository
git clone https://github.com/mardon3/roulettify.git
cd roulettify

# Set up environment variables
cp .env.example .env
# Edit .env with your Spotify credentials

# Run with hot reload (installs air if needed)
make watch

# Or run manually
make run
```

### Docker

```bash
# Build and run with Docker Compose
make docker-run

# Access at http://127.0.0.1:8080
```

### Testing

```bash
# Run all tests
make test

# Run with coverage
make test-coverage

# Test for race conditions
make test-race
```

## 🔧 Configuration

### Environment Variables

Create a `.env` file in the root directory:

```bash
# Server
PORT=8080
APP_ENV=development

# Frontend
FRONTEND_URL=http://127.0.0.1:5173

# Spotify (Get from https://developer.spotify.com/dashboard)
SPOTIFY_CLIENT_ID=your_client_id_here
SPOTIFY_CLIENT_SECRET=your_client_secret_here
SPOTIFY_REDIRECT_URI=http://127.0.0.1:8080/auth/callback

# CORS
ALLOWED_ORIGINS=http://127.0.0.1:3000,http://127.0.0.1:5173

# Game Settings
DEFAULT_TOTAL_ROUNDS=10
MAX_PLAYERS_PER_ROOM=10
```

### Spotify Developer Setup

1. Go to [Spotify Developer Dashboard](https://developer.spotify.com/dashboard)
2. Create a new app
3. Add redirect URI: `http://127.0.0.1:8080/auth/callback`
4. Copy Client ID and Client Secret to `.env`
5. For production, add your production URL to redirect URIs

## 🏗️ Architecture

### Persistent Room System
- **3 fixed rooms** (Room 1, Room 2, Room 3) created at startup
- **10 player capacity** per room (30 total concurrent players)
- **No dynamic creation/deletion** - rooms never shut down
- **Memory efficient** - optimized for low-memory cloud deployments
- **Consistent ordering** - rooms always appear in the same order in UI

### Preview URL Strategy
As of Nov 2024, Spotify no longer provides preview URLs via API for new applications. This project uses web scraping:
- Fetches Spotify embed pages for each track
- Extracts preview URLs using regex patterns
- Implements caching to reduce scraping overhead
- Falls back gracefully when previews unavailable

## 📊 Performance

- **Memory**: 3 persistent rooms
- **Concurrent Users**: Up to 30 (10 per room)
- **WebSocket Connections**: Persistent per player
- **Latency**: <100ms for game actions
- **Deployment**: Minimal resource usage 