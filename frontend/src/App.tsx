import { useState, useEffect } from 'react'

interface Player {
  id: string
  name: string
  spotify_id: string
  access_token?: string
}

function App() {
  const [count, setCount] = useState(0)
  const [message, setMessage] = useState<string>('')
  const [player, setPlayer] = useState<Player | null>(null)
  const [isAuthenticated, setIsAuthenticated] = useState(false)

  // Helper function to get cookie value
  const getCookie = (name: string): string | null => {
    const value = `; ${document.cookie}`
    const parts = value.split(`; ${name}=`)
    if (parts.length === 2) {
      return parts.pop()?.split(';').shift() || null
    }
    return null
  }

  // Check for OAuth callback on mount
  useEffect(() => {
    const checkAuth = () => {
      console.log('App mounted, checking for auth...')
      const urlParams = new URLSearchParams(window.location.search)
      const authSuccess = urlParams.get('auth')
      
      console.log('URL params:', { authSuccess })
      
      if (authSuccess === 'success') {
        console.log('Auth success detected, clearing URL...')
        // Clear URL parameters
        window.history.replaceState({}, document.title, '/')
        
        // Try to get player data from cookie
        console.log('All cookies:', document.cookie)
        const playerData = getCookie('player_session')
        console.log('Player session cookie:', playerData)
        
        if (playerData) {
          try {
            const parsed = JSON.parse(decodeURIComponent(playerData))
            
            console.log('Parsed player data:', parsed)
            setPlayer({
              id: parsed.id,
              name: parsed.name,
              spotify_id: parsed.spotify_id,
            })
            setIsAuthenticated(true)
            console.log('Authentication state updated!')
          } catch (e) {
            console.error('Failed to parse player data:', e)
          }
        } else {
          console.error('No player_session cookie found!')
        }
      }
    }

    checkAuth()
  }, [])

  const fetchData = () => {
    fetch(`http://localhost:${import.meta.env.VITE_PORT || 8080}/`)
      .then(response => response.json())
      .then(data => setMessage(data.message))
      .catch(error => console.error('Error fetching data:', error))
  }

  const handleSpotifyAuth = () => {
    window.location.href = `http://localhost:${import.meta.env.VITE_PORT || 8080}/auth/spotify`
  }

  return (
    <div className="min-h-screen bg-gray-50 py-12 px-4 sm:px-6 lg:px-8">
      <div className="max-w-md mx-auto space-y-8">
        <div className="text-center">
          <h1 className="text-4xl font-bold text-gray-900 mb-2">
            🎵 Roulettify
          </h1>
          <p className="text-gray-600">
            Spotify Multiplayer Music Game
          </p>
        </div>

        <div className="bg-white p-6 rounded-lg shadow-md">
          {!isAuthenticated ? (
            <div className="text-center space-y-4">
              <h2 className="text-xl font-semibold text-gray-800">
                Sign in with Spotify
              </h2>
              <p className="text-gray-600 text-sm">
                Connect your Spotify account to start playing
              </p>
              <button
                onClick={handleSpotifyAuth}
                className="w-full bg-green-500 hover:bg-green-600 text-white font-semibold py-3 px-4 rounded-md transition-colors"
              >
                🎵 Sign in with Spotify
              </button>
            </div>
          ) : (
            <div className="space-y-4">
              <div className="text-center">
                <h2 className="text-xl font-semibold text-gray-800">
                  Welcome, {player?.name}! ✓
                </h2>
                <p className="text-green-600 text-sm mt-2">
                  Successfully authenticated with Spotify
                </p>
              </div>
              
              <div className="border-t pt-4">
                <button
                  onClick={() => setCount((count) => count + 1)}
                  className="w-full bg-blue-500 hover:bg-blue-600 text-white font-semibold py-2 px-4 rounded-md transition-colors mb-2"
                >
                  Count is {count}
                </button>
                
                <button
                  onClick={fetchData}
                  className="w-full bg-gray-500 hover:bg-gray-600 text-white font-semibold py-2 px-4 rounded-md transition-colors"
                >
                  Fetch from Server
                </button>

                {message && (
                  <div className="mt-4 p-4 bg-gray-50 rounded-md">
                    <p className="text-gray-700 text-sm">Server Response:</p>
                    <p className="text-gray-900 font-medium">{message}</p>
                  </div>
                )}
              </div>
            </div>
          )}
        </div>

        <div className="text-center text-gray-500 text-sm">
          <p>Built with Go, React, and Tailwind CSS</p>
          <p className="mt-2 text-xs">Gin • Spotify API • WebSockets</p>
        </div>
      </div>
    </div>
  )
}

export default App