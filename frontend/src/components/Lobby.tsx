import { useState } from 'react'

interface Player {
  id: string
  name: string
  is_guest?: boolean
}

interface LobbyProps {
  player: Player | null
  isAuthenticated: boolean
  onJoinRoom: (roomId: string) => void
  onGuestLogin: (guestIndex: number) => void
}

export default function Lobby({ player, isAuthenticated, onJoinRoom, onGuestLogin }: LobbyProps) {
  const [roomInput, setRoomInput] = useState('')
  const [guestIndex, setGuestIndex] = useState(0)

  const handleSpotifyAuth = () => {
    window.location.href = 'http://localhost:8080/auth/spotify'
  }

  const handleGuestClick = () => {
    onGuestLogin(guestIndex)
    setGuestIndex(guestIndex + 1)
  }

  const handleCreateRoom = () => {
    if (roomInput.trim()) {
      onJoinRoom(roomInput.trim())
    }
  }

  const handleJoinRoom = () => {
    if (roomInput.trim()) {
      onJoinRoom(roomInput.trim())
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
                  Connect with Spotify or play as a guest
                </p>
              </div>

              <button
                onClick={handleSpotifyAuth}
                className="w-full bg-green-500 hover:bg-green-600 text-white font-bold py-4 px-6 rounded-xl transition-all transform hover:scale-105 shadow-lg"
              >
                <span className="text-xl">🎵 Sign in with Spotify</span>
              </button>

              <div className="relative">
                <div className="absolute inset-0 flex items-center">
                  <div className="w-full border-t border-gray-300"></div>
                </div>
                <div className="relative flex justify-center text-sm">
                  <span className="px-2 bg-white text-gray-500">OR</span>
                </div>
              </div>

              <button
                onClick={handleGuestClick}
                className="w-full bg-purple-500 hover:bg-purple-600 text-white font-bold py-4 px-6 rounded-xl transition-all transform hover:scale-105 shadow-lg"
              >
                <span className="text-xl">👤 Continue as Guest</span>
              </button>

              <div className="bg-blue-50 border border-blue-200 rounded-lg p-4">
                <p className="text-sm text-blue-800">
                  <strong>Guest Mode:</strong> Test the game with mock data without needing multiple Spotify accounts!
                </p>
              </div>
            </div>
          ) : (
            <div className="space-y-6">
              <div className="text-center pb-4 border-b">
                <h2 className="text-2xl font-bold text-gray-800 mb-2">
                  Welcome, {player?.name}! {player?.is_guest ? '👤' : '✓'}
                </h2>
                <p className="text-sm text-gray-600">
                  {player?.is_guest ? 'Playing as Guest' : 'Connected with Spotify'}
                </p>
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Room Name
                </label>
                <input
                  type="text"
                  placeholder="Enter room name..."
                  value={roomInput}
                  onChange={(e) => setRoomInput(e.target.value)}
                  onKeyPress={(e) => e.key === 'Enter' && handleCreateRoom()}
                  className="w-full px-4 py-3 border-2 border-gray-300 rounded-lg focus:ring-2 focus:ring-purple-500 focus:border-transparent"
                />
              </div>

              <div className="grid grid-cols-2 gap-4">
                <button
                  onClick={handleCreateRoom}
                  disabled={!roomInput.trim()}
                  className="bg-purple-500 hover:bg-purple-600 disabled:bg-gray-300 disabled:cursor-not-allowed text-white font-bold py-3 px-6 rounded-lg transition-all transform hover:scale-105 shadow-md"
                >
                  Create Room
                </button>
                <button
                  onClick={handleJoinRoom}
                  disabled={!roomInput.trim()}
                  className="bg-pink-500 hover:bg-pink-600 disabled:bg-gray-300 disabled:cursor-not-allowed text-white font-bold py-3 px-6 rounded-lg transition-all transform hover:scale-105 shadow-md"
                >
                  Join Room
                </button>
              </div>
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
