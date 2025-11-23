import { useState, useEffect } from 'react'
import Lobby from './components/Lobby'
import GameRoom from './components/GameRoom'

interface Player {
  id: string
  name: string
  spotify_id: string
  access_token?: string
}

function App() {
  const [player, setPlayer] = useState<Player | null>(null)
  const [isAuthenticated, setIsAuthenticated] = useState(false)
  const [gameState, setGameState] = useState<'lobby' | 'room'>('lobby')
  const [roomId, setRoomId] = useState('')

  const getCookie = (name: string): string | null => {
    const value = `; ${document.cookie}`
    const parts = value.split(`; ${name}=`)
    if (parts.length === 2) {
      return parts.pop()?.split(';').shift() || null
    }
    return null
  }

  useEffect(() => {
    const checkAuth = () => {
      const urlParams = new URLSearchParams(window.location.search)
      const authSuccess = urlParams.get('auth')
      
      if (authSuccess === 'success') {
        window.history.replaceState({}, document.title, '/')
        
        const playerData = getCookie('player_session')
        
        if (playerData) {
          try {
            const parsed = JSON.parse(decodeURIComponent(playerData))
          
            setPlayer({
              id: parsed.id,
              name: parsed.name,
              spotify_id: parsed.spotify_id,
              access_token: parsed.access_token,
            })
            setIsAuthenticated(true)
          } catch (e) {
            console.error('Failed to parse player data:', e)
          }
        }
      }
    }

    checkAuth()
  }, [])

  const handleJoinRoom = (room: string) => {
    setRoomId(room)
    setGameState('room')
  }

  const handleLeaveRoom = () => {
    setGameState('lobby')
    setRoomId('')
  }

  return (
    <div className="min-h-screen bg-linear-to-br from-purple-600 via-pink-500 to-red-500">
      {gameState === 'lobby' ? (
        <Lobby
          player={player}
          isAuthenticated={isAuthenticated}
          onJoinRoom={handleJoinRoom}
        />
      ) : (
        player && (
          <GameRoom
            roomId={roomId}
            player={player}
            onLeaveRoom={handleLeaveRoom}
          />
        )
      )}
    </div>
  )
}

export default App