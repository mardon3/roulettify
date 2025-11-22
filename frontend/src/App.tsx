import { useState, useEffect } from 'react'
import Lobby from './components/Lobby'
import GameRoom from './components/GameRoom'

interface Player {
  id: string
  name: string
  spotify_id: string
  access_token?: string
  is_guest?: boolean
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
      console.log('App mounted, checking for auth...')
      const urlParams = new URLSearchParams(window.location.search)
      const authSuccess = urlParams.get('auth')
      
      if (authSuccess === 'success') {
        console.log('Auth success detected, clearing URL...')
        window.history.replaceState({}, document.title, '/')
        
        const playerData = getCookie('player_session')
        console.log('Player session cookie:', playerData)
        
        if (playerData) {
          try {
            const parsed = JSON.parse(decodeURIComponent(playerData))
          
            setPlayer({
              id: parsed.id,
              name: parsed.name,
              spotify_id: parsed.spotify_id,
              access_token: parsed.access_token,
              is_guest: parsed.is_guest || false,
            })
            setIsAuthenticated(true)
            console.log('Authentication state updated!')
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

  const handleGuestLogin = async (guestIndex: number) => {
    try {
      console.log('Requesting guest login with index:', guestIndex)
      
      const response = await fetch(`http://localhost:8080/auth/guest`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ guest_index: guestIndex }),
      })

      const data = await response.json()
      console.log('Guest login response:', data)
      
      if (data.success) {
        const parsed = JSON.parse(data.player_data)
        console.log('Guest player created:', parsed)
        
        setPlayer({
          id: parsed.id,
          name: parsed.name,
          spotify_id: parsed.spotify_id,
          access_token: parsed.access_token,
          is_guest: true,
        })
        setIsAuthenticated(true)
      }
    } catch (error) {
      console.error('Guest login error:', error)
    }
  }

  return (
    <div className="min-h-screen bg-linear-to-br from-purple-600 via-pink-500 to-red-500">
      {gameState === 'lobby' ? (
        <Lobby
          player={player}
          isAuthenticated={isAuthenticated}
          onJoinRoom={handleJoinRoom}
          onGuestLogin={handleGuestLogin}
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