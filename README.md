# Roulettify - Spotify Multiplayer Game

A real-time multiplayer web game where players authenticate with Spotify and guess which friend has listened to randomly selected tracks the most based on their top 50 tracks from the past 6 months.

## Features

- **Spotify Integration**: OAuth2 authentication with zmb3/spotify library
- **Real-time Multiplayer**: WebSocket-based game communication
- **30-Second Previews**: Each round plays a preview snippet automatically
- **Scoring System**: 10 points for correct guess + 5 points speed bonus for fastest
- **In-Memory Architecture**: Optimized for small-scale gameplay (3-10 players)

## Technology Stack

- **Backend**: Go 1.25+ with Gin web framework
- **Frontend**: React 18 + TypeScript with Vite
- **Real-time Communication**: WebSocket with gorilla/websocket
- **Spotify API**: zmb3/spotify/v2 library with OAuth2

## Prerequisites

- Go 1.25 or higher
- Node.js 18+ and npm
- Spotify Developer account and application credentials

## Project Structure

```
roulettify/
├── cmd/
│   └── api/
│       └── main.go              # Application entry point
├── internal/
│   ├── server/
│   │   ├── server.go            # Server initialization
│   │   └── routes.go            # Chi router + Spotify OAuth handlers
│   └── auth/
│       └── spotify.go           # Spotify authentication logic
├── frontend/
│   ├── src/
│   │   ├── App.tsx              # Main React component
│   │   └── main.tsx             # React entry point
│   └── package.json
├── Dockerfile                    # Multi-stage Docker build
├── docker-compose.yml           # Docker Compose configuration
├── Makefile                     # Build automation
├── go.mod                       # Go dependencies
└── .env                         # Environment variables
```

## Environment Variables

```bash
# Server
PORT=8080

# Frontend URL (for OAuth redirect)
FRONTEND_URL=http://localhost:5173

# Spotify OAuth
SPOTIFY_CLIENT_ID=PLACEHOLDER_CLIENT_ID
SPOTIFY_CLIENT_SECRET=PLACEHOLDER_CLIENT_SECRET
SPOTIFY_REDIRECT_URI=http://localhost:8080/auth/callback

# CORS
ALLOWED_ORIGINS=http://localhost:3000,http://localhost:5173

# Game Settings
DEFAULT_TOTAL_ROUNDS=10
MAX_PLAYERS_PER_ROOM=5
```

## API Endpoints

### REST

- `GET /` - Health check
- `GET /health` - Detailed health status
- `GET /auth/spotify` - Initiate Spotify OAuth flow
- `GET /auth/callback` - OAuth callback handler

### WebSocket

- `GET /websocket` - WebSocket connection for real-time game updates

## WebSocket Messages

### Client → Server

**Join Room**
```json
{ "type": "join_room", "payload": { "room_id": "...", "player_id": "...", "access_token": "..." } }
```

**Start Game**
```json
{ "type": "start_game", "payload": { "room_id": "...", "total_rounds": 10 } }
```

**Submit Guess**
```json
{ "type": "submit_guess", "payload": { "room_id": "...", "player_id": "...", "guessed_player_id": "..." } }
```

### Server → Client

**Player Joined**
```json
{ "type": "player_joined", "payload": { "player": {...}, "player_count": 2 } }
```

**Round Started**
```json
{ "type": "round_started", "payload": { "round": 1, "total_rounds": 10, "track": {...}, "players": [...] } }
```

**Round Complete**
```json
{ "type": "round_complete", "payload": { "winner_id": "...", "points_awarded": {...}, "updated_scores": {...} } }
```

**Game Over**
```json
{ "type": "game_over", "payload": { "winner_id": "...", "final_scores": {...} } }
```

## Development Tips

### Hot Reload Go

Install and use `air` for auto-reload:
```bash
go install github.com/cosmtrek/air@latest
air
```

### Test WebSocket

In browser console:
```javascript
const ws = new WebSocket('ws://localhost:8080/ws');
ws.onmessage = (e) => console.log(JSON.parse(e.data));
ws.send(JSON.stringify({
  type: 'join_room',
  payload: { room_id: 'test', player_id: 'p1', access_token: 'token' }
}));
```

## Important Notes

### Spotify Preview URLs

As of November 2024, Spotify deprecated preview URLs for new applications. If preview URLs are unavailable:

1. The game will attempt to find tracks with previews
2. Falls back to using Spotify embed player
3. Consider applying for Extended API Access with Spotify


## Scoring Rules

- **Base Points**: 10 points for correct guess
- **Speed Bonus**: 5 additional points for fastest guess
- **Track Ranking**: Winner is determined by who has track ranked highest (lowest rank #)
