import { useState, useEffect } from 'react'

interface Player {
  id: string
  name: string
}

interface Room {
  id: string
  player_count: number
  max_players: number
  state: string
}

interface LobbyProps {
  player: Player | null
  isAuthenticated: boolean
  onJoinRoom: (roomId: string) => void
}

export default function Lobby({ player, isAuthenticated, onJoinRoom }: LobbyProps) {
  const [rooms, setRooms] = useState<Room[]>([])
  const [isJoining, setIsJoining] = useState(false)
  const [joinError, setJoinError] = useState<string | null>(null)

  useEffect(() => {
    if (!isAuthenticated) return

    const fetchRooms = async () => {
      try {
        const response = await fetch('http://localhost:8080/rooms')
        const data = await response.json()
        setRooms(data.rooms || [])
      } catch (error) {
        console.error('Failed to fetch rooms:', error)
      }
    }

    fetchRooms()
    // Refresh room list every 3 seconds
    const interval = setInterval(fetchRooms, 3000)
    return () => clearInterval(interval)
  }, [isAuthenticated])

  const handleSpotifyAuth = () => {
    window.location.href = 'http://localhost:8080/auth/spotify'
  }

  const handleJoinRoom = (roomId: string) => {
    if (!isJoining) {
      setIsJoining(true)
      setJoinError(null)
      
      // Small delay for visual feedback
      setTimeout(() => {
        onJoinRoom(roomId)
        setIsJoining(false)
      }, 300)
    }
  }

  const getStateLabel = (state: string) => {
    switch (state) {
      case 'waiting':
        return { text: 'Waiting', color: 'bg-green-100 text-green-800' }
      case 'playing':
        return { text: 'In Game', color: 'bg-yellow-100 text-yellow-800' }
      case 'game_over':
        return { text: 'Game Over', color: 'bg-blue-100 text-blue-800' }
      default:
        return { text: 'Unknown', color: 'bg-gray-100 text-gray-800' }
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center p-4">
      <div className="max-w-2xl w-full">
        <div className="text-center mb-8">
          <h1 className="text-6xl font-bold text-white mb-4 drop-shadow-lg">
            🎵 Roulettify
          </h1>
          <p className="text-xl text-white/90 drop-shadow-md">
            Guess which friend listened to each track the most!
          </p>
        </div>

        <div className="bg-white rounded-2xl shadow-2xl p-8 mb-6">
          {!isAuthenticated ? (
            <div className="space-y-6">
              <div className="text-center">
                <h2 className="text-2xl font-bold text-gray-800 mb-2">
                  Sign In
                </h2>
                <p className="text-gray-600">
                  Connect with Spotify to play
                </p>
              </div>

              <button
                onClick={handleSpotifyAuth}
                className="w-full bg-green-500 hover:bg-green-600 text-white font-bold py-4 px-6 rounded-xl transition-all transform hover:scale-105 shadow-lg"
              >
                <span className="text-xl">🎵 Sign in with Spotify</span>
              </button>

              <div className="bg-yellow-50 border border-yellow-300 rounded-lg p-4">
                <p className="text-sm text-yellow-800">
                  ℹ️ <strong>Note:</strong> It may take up to a minute after authentication to fetch your top 50 Spotify tracks.
                </p>
              </div>

              <div className="bg-blue-50 border border-blue-200 rounded-lg p-4">
                <p className="text-sm text-blue-800">
                  🔐 We securely access your top 50 Spotify tracks to play the game.
                </p>
              </div>
            </div>
          ) : (
            <div className="space-y-6">
              <div className="text-center pb-4 border-b">
                <h2 className="text-2xl font-bold text-gray-800 mb-2">
                  Welcome, {player?.name}! ✓
                </h2>
                <p className="text-sm text-gray-600">
                  Connected with Spotify
                </p>
              </div>

              <div>
                <h3 className="text-lg font-semibold text-gray-800 mb-3">
                  Select a Room
                </h3>
                
                {rooms.length === 0 ? (
                  <div className="text-center py-8">
                    <div className="animate-spin text-4xl mb-2">⏳</div>
                    <p className="text-gray-600">Loading rooms...</p>
                  </div>
                ) : (
                  <div className="space-y-3">
                    {rooms.map((room) => {
                      const stateInfo = getStateLabel(room.state)
                      const isFull = room.player_count >= room.max_players
                      return (
                        <button
                          key={room.id}
                          onClick={() => handleJoinRoom(room.id)}
                          disabled={isJoining || isFull}
                          className={`w-full rounded-xl p-4 transition-all transform shadow-lg ${
                            isFull
                              ? 'bg-gray-400 cursor-not-allowed opacity-60'
                              : 'bg-linear-to-r from-purple-500 to-pink-500 hover:from-purple-600 hover:to-pink-600 hover:scale-102'
                          } text-white disabled:cursor-not-allowed`}
                        >
                          <div className="flex items-center justify-between">
                            <div className="flex items-center gap-3">
                              <span className="text-2xl">🎮</span>
                              <div className="text-left">
                                <p className="font-bold text-lg">{room.id}</p>
                                <p className="text-sm opacity-90">
                                  {room.player_count}/{room.max_players} players {isFull && '(Full)'}
                                </p>
                              </div>
                            </div>
                            <span className={`px-3 py-1 rounded-full text-xs font-semibold ${stateInfo.color}`}>
                              {stateInfo.text}
                            </span>
                          </div>
                        </button>
                      )
                    })}
                  </div>
                )}
              </div>

              {joinError && (
                <div className="bg-red-50 border border-red-300 rounded-lg p-4">
                  <div className="flex items-start justify-between">
                    <div className="flex items-start gap-2">
                      <span className="text-red-600">⚠️</span>
                      <p className="text-sm text-red-700">{joinError}</p>
                    </div>
                    <button
                      onClick={() => setJoinError(null)}
                      className="text-red-600 hover:text-red-800 font-bold"
                    >
                      ✕
                    </button>
                  </div>
                </div>
              )}
            </div>
          )}
        </div>

        <div className="bg-white/10 backdrop-blur-sm rounded-xl p-6 text-white">
          <h3 className="font-bold text-lg mb-3">How to Play:</h3>
          <ul className="space-y-2 text-sm">
            <li>🎵 Listen to 30-second track previews</li>
            <li>👥 Guess which player has the track ranked highest</li>
            <li>⭐ Earn 10 points for correct guesses</li>
            <li>⚡ Get 5 bonus points for being the fastest correct guesser</li>
            <li>🏆 Player with most points wins!</li>
          </ul>
        </div>
      </div>
    </div>
  )
}