import { useState } from 'react'

interface Player {
  id: string
  name: string
}

interface LobbyProps {
  player: Player | null
  isAuthenticated: boolean
  onJoinRoom: (roomId: string) => void
}

export default function Lobby({ player, isAuthenticated, onJoinRoom }: LobbyProps) {
  const [roomInput, setRoomInput] = useState('')
  const [isJoining, setIsJoining] = useState(false)

  const handleSpotifyAuth = () => {
    window.location.href = 'http://localhost:8080/auth/spotify'
  }

  const handleJoinOrCreateRoom = () => {
    if (roomInput.trim() && !isJoining) {
      setIsJoining(true)
      setTimeout(() => {
        onJoinRoom(roomInput.trim())
        setIsJoining(false)
      }, 300)
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
                  ℹ️ <strong>Note:</strong> It may take a few seconds after authentication to fetch your top 50 Spotify tracks.
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
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Room Name
                </label>
                <input
                  type="text"
                  placeholder="Enter room name..."
                  value={roomInput}
                  onChange={(e) => setRoomInput(e.target.value)}
                  onKeyPress={(e) => e.key === 'Enter' && handleJoinOrCreateRoom()}
                  className="w-full px-4 py-3 border-2 border-gray-300 rounded-lg focus:ring-2 focus:ring-purple-500 focus:border-transparent transition-all"
                />
                <p className="text-xs text-gray-500 mt-1">
                  Enter any room name to create or join
                </p>
              </div>

              <button
                onClick={handleJoinOrCreateRoom}
                disabled={!roomInput.trim() || isJoining}
                className={`w-full font-bold py-4 px-6 rounded-xl transition-all transform shadow-lg ${
                  !roomInput.trim() || isJoining
                    ? 'bg-gray-300 cursor-not-allowed'
                    : 'bg-linear-to-r from-purple-500 to-pink-500 hover:from-purple-600 hover:to-pink-600 hover:scale-105 text-white'
                }`}
              >
                {isJoining ? (
                  <span className="flex items-center justify-center gap-2">
                    <span className="animate-spin">⏳</span> Joining...
                  </span>
                ) : (
                  <span className="text-xl">🚀 Join / Create Room</span>
                )}
              </button>
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