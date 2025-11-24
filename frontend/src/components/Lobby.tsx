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
        const response = await fetch('/rooms')
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
    window.location.href = '/auth/spotify'
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
        return { text: 'Waiting', color: 'bg-green-500/20 text-green-400 border border-green-500/30' }
      case 'playing':
        return { text: 'In Game', color: 'bg-yellow-500/20 text-yellow-400 border border-yellow-500/30' }
      case 'game_over':
        return { text: 'Game Over', color: 'bg-blue-500/20 text-blue-400 border border-blue-500/30' }
      default:
        return { text: 'Unknown', color: 'bg-gray-500/20 text-gray-400 border border-gray-500/30' }
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center p-4">
      <div className="max-w-2xl w-full">
        <div className="text-center mb-12 animate-float">
          <h1 className="text-5xl md:text-7xl font-bold text-white mb-4 tracking-tight">
            <span className="text-spotify-green">Roulett</span>ify
          </h1>
          <p className="text-lg md:text-xl text-gray-400 font-light">
            How well do you know your friends' music taste?
          </p>
        </div>

        <div className="glass-panel rounded-3xl p-6 md:p-8 mb-8 shadow-2xl">
          {!isAuthenticated ? (
            <div className="space-y-8 py-4">
              <div className="text-center">
                <h2 className="text-3xl font-bold text-white mb-3">
                  Ready to Play?
                </h2>
                <p className="text-gray-400">
                  Connect your Spotify account to get started
                </p>
              </div>

              <button
                onClick={handleSpotifyAuth}
                className="w-full bg-spotify-green hover:bg-[#1ed760] text-black font-bold py-4 px-8 rounded-full transition-all transform hover:scale-105 shadow-[0_0_20px_rgba(29,185,84,0.3)] flex items-center justify-center gap-3"
              >
                <svg className="w-8 h-8" viewBox="0 0 24 24" fill="currentColor">
                  <path d="M12 0C5.4 0 0 5.4 0 12s5.4 12 12 12 12-5.4 12-12S18.66 0 12 0zm5.521 17.34c-.24.359-.66.48-1.021.24-2.82-1.74-6.36-2.101-10.561-1.141-.418.122-.779-.179-.899-.539-.12-.421.18-.78.54-.9 4.56-1.021 8.52-.6 11.64 1.32.42.18.479.659.301 1.02zm1.44-3.3c-.301.42-.841.6-1.262.3-3.239-1.98-8.159-2.58-11.939-1.38-.479.12-1.02-.12-1.14-.6-.12-.48.12-1.021.6-1.141 4.32-1.38 9.841-.719 13.44 1.56.42.3.6.84.3 1.26zm.12-3.36C14.939 7.98 8.699 7.8 5.1 8.88c-.6.18-1.2-.18-1.38-.72-.18-.6.18-1.2.72-1.38 4.139-1.26 10.98-1.08 15.481 1.56.539.3.719.96.42 1.5-.3.54-.96.72-1.5.42z"/>
                </svg>
                <span className="text-xl tracking-wide">Continue with Spotify</span>
              </button>

              <div className="grid grid-cols-1 gap-4 text-sm">
                <div className="bg-yellow-500/10 border border-yellow-500/20 rounded-xl p-4 flex gap-3">
                  <span className="text-xl">⏳</span>
                  <p className="text-yellow-200/80">
                    It may take a moment to fetch your top tracks after signing in.
                  </p>
                </div>

                <div className="bg-blue-500/10 border border-blue-500/20 rounded-xl p-4 flex gap-3">
                  <span className="text-xl">🔒</span>
                  <p className="text-blue-200/80">
                    We only access your top 50 tracks to generate quiz questions.
                  </p>
                </div>
              </div>
            </div>
          ) : (
            <div className="space-y-6">
              <div className="flex items-center justify-between pb-6 border-b border-white/10">
                <div>
                  <h2 className="text-2xl font-bold text-white mb-1">
                    Welcome back, {player?.name}
                  </h2>
                  <div className="flex items-center gap-2 text-green-400 text-sm">
                    <span className="w-2 h-2 bg-green-500 rounded-full animate-pulse"></span>
                    Connected to Spotify
                  </div>
                </div>
                <div className="w-12 h-12 bg-spotify-light-gray rounded-full flex items-center justify-center text-2xl border border-white/10">
                  👤
                </div>
              </div>

              <div>
                <div className="flex justify-between items-end mb-4">
                  <h3 className="text-lg font-semibold text-gray-300">
                    Available Rooms
                  </h3>
                  <span className="text-xs text-gray-500 uppercase tracking-wider font-bold">
                    Live Now
                  </span>
                </div>
                
                {rooms.length === 0 ? (
                  <div className="text-center py-12 bg-spotify-light-gray/50 rounded-2xl border border-white/5">
                    <div className="animate-spin text-4xl mb-3 text-spotify-green">◌</div>
                    <p className="text-gray-400">Searching for active rooms...</p>
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
                          className={`w-full group relative overflow-hidden rounded-xl p-4 transition-all border border-white/5 ${
                            isFull
                              ? 'bg-gray-800/50 cursor-not-allowed opacity-60'
                              : 'bg-spotify-light-gray hover:bg-[#333] hover:border-white/20 hover:scale-[1.02] hover:shadow-xl'
                          }`}
                        >
                          <div className="flex items-center justify-between relative z-10">
                            <div className="flex items-center gap-4">
                              <div className={`w-12 h-12 rounded-lg flex items-center justify-center text-2xl ${
                                isFull ? 'bg-gray-700' : 'bg-linear-to-br from-purple-600 to-blue-600'
                              }`}>
                                🎵
                              </div>
                              <div className="text-left">
                                <p className="font-bold text-lg text-white group-hover:text-spotify-green transition-colors">
                                  {room.id}
                                </p>
                                <p className="text-sm text-gray-400">
                                  {room.player_count}/{room.max_players} players
                                </p>
                              </div>
                            </div>
                            <span className={`px-3 py-1 rounded-full text-xs font-bold uppercase tracking-wider ${stateInfo.color}`}>
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
                <div className="bg-red-500/10 border border-red-500/20 rounded-xl p-4 animate-shake">
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-3">
                      <span className="text-red-400">⚠️</span>
                      <p className="text-sm text-red-200">{joinError}</p>
                    </div>
                    <button
                      onClick={() => setJoinError(null)}
                      className="text-red-400 hover:text-red-200"
                    >
                      ✕
                    </button>
                  </div>
                </div>
              )}
            </div>
          )}
        </div>

        <div className="glass-panel rounded-2xl p-6 text-gray-300 text-sm">
          <h3 className="font-bold text-white mb-4 flex items-center gap-2">
            <span className="text-xl">💡</span> How to Play
          </h3>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div className="flex items-start gap-3">
              <span className="bg-white/10 w-6 h-6 rounded-full flex items-center justify-center text-xs font-bold text-spotify-green">1</span>
              <p>Listen to 30-second track previews from everyone's top songs</p>
            </div>
            <div className="flex items-start gap-3">
              <span className="bg-white/10 w-6 h-6 rounded-full flex items-center justify-center text-xs font-bold text-spotify-green">2</span>
              <p>Guess which friend has the track in their top 50</p>
            </div>
            <div className="flex items-start gap-3">
              <span className="bg-white/10 w-6 h-6 rounded-full flex items-center justify-center text-xs font-bold text-spotify-green">3</span>
              <p>Earn points for speed and accuracy</p>
            </div>
            <div className="flex items-start gap-3">
              <span className="bg-white/10 w-6 h-6 rounded-full flex items-center justify-center text-xs font-bold text-spotify-green">4</span>
              <p>Win by knowing your friends' music taste best!</p>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}

