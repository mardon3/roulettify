import { useState, useEffect, useRef } from 'react'

interface Player {
  id: string
  name: string
  access_token?: string
}

interface PlayerInfo {
  id: string
  name: string
  score: number
}

interface Track {
  id: string
  name: string
  artists: string[]
  image_url: string
  preview_url: string
}

interface GameRoomProps {
  roomId: string
  player: Player
  onLeaveRoom: () => void
}

interface RoundResult {
  round: number
  track: Track
  winner_id: string
  winner_rank: number
  correct_guessers: string[]
  points_awarded: Record<string, number>
  all_rankings: Record<string, number>
  updated_scores: Record<string, number>
}

export default function GameRoom({ roomId, player, onLeaveRoom }: GameRoomProps) {
  const wsRef = useRef<WebSocket | null>(null)
  const hasConnected = useRef(false)
  
  const [players, setPlayers] = useState<PlayerInfo[]>([])
  const [gameState, setGameState] = useState<'waiting' | 'playing' | 'round_end' | 'game_over'>('waiting')
  const [currentRound, setCurrentRound] = useState(0)
  const [totalRounds, setTotalRounds] = useState(10)
  const [currentTrack, setCurrentTrack] = useState<Track | null>(null)
  const [hasGuessed, setHasGuessed] = useState(false)
  const [guessesCount, setGuessesCount] = useState(0)
  const [roundResult, setRoundResult] = useState<RoundResult | null>(null)
  const [timeRemaining, setTimeRemaining] = useState(30)
  const [isStarting, setIsStarting] = useState(false)
  const audioRef = useRef<HTMLAudioElement>(null)
  const [audioError, setAudioError] = useState<string | null>(null)
  const [errorMessage, setErrorMessage] = useState<string | null>(null)

  useEffect(() => {
    if (hasConnected.current) return
    hasConnected.current = true

    const websocket = new WebSocket('ws://localhost:8080/ws')
    wsRef.current = websocket

    websocket.onopen = () => {
      console.log('WebSocket connected')
      
      websocket.send(JSON.stringify({
        type: 'join_room',
        payload: {
          room_id: roomId,
          player_id: player.id,
          player_name: player.name,
          access_token: player.access_token || '',
        },
      }))
    }

    websocket.onmessage = (event) => {
      const message = JSON.parse(event.data)
      console.log('WebSocket message:', message)

      switch (message.type) {
        case 'player_joined':
          setPlayers(message.payload.players || [])
          break

        case 'player_left':
          setPlayers(message.payload.players || [])
          break

        case 'game_started':
          setGameState('waiting')
          setTotalRounds(message.payload.total_rounds)
          setIsStarting(false)
          break

        case 'round_started':
          setGameState('playing')
          setCurrentRound(message.payload.round)
          setTotalRounds(message.payload.total_rounds)
          setCurrentTrack(message.payload.track)
          setHasGuessed(false)
          setGuessesCount(0)
          setRoundResult(null)
          setTimeRemaining(30)
          setAudioError(null)
          
          if (message.payload.track.preview_url && audioRef.current) {
            audioRef.current.src = message.payload.track.preview_url
            audioRef.current.volume = 0.7
            audioRef.current.play().catch(err => {
              console.log('Audio play error:', err)
              setAudioError('Failed to play audio preview')
            })
          }
          break

        case 'guess_received':
          setGuessesCount(message.payload.guesses_count)
          break

        case 'round_complete':
          setGameState('round_end')
          setRoundResult(message.payload)
          setPlayers(prev => prev.map(p => ({
            ...p,
            score: message.payload.updated_scores[p.id] || 0
          })))
          
          if (audioRef.current) {
            audioRef.current.pause()
          }
          break

        case 'game_over':
          setGameState('game_over')
          setPlayers(prev => prev.map(p => ({
            ...p,
            score: message.payload.final_scores[p.id] || 0
          })))
          break

        case 'error':
          console.error('Game error:', message.payload.message)
          setErrorMessage(message.payload.message)
          setIsStarting(false)
          
          // Auto-clear errors after 5 seconds
          setTimeout(() => setErrorMessage(null), 5000)
          break
      }
    }

    websocket.onerror = (error) => {
      console.error('WebSocket error:', error)
      setIsStarting(false)
    }

    websocket.onclose = () => {
      console.log('WebSocket disconnected')
    }

    return () => {
      if (websocket.readyState === WebSocket.OPEN) {
        websocket.close()
      }
    }
  }, [roomId, player])

  useEffect(() => {
    if (gameState === 'playing' && timeRemaining > 0) {
      const timer = setTimeout(() => {
        setTimeRemaining(timeRemaining - 1)
      }, 1000)
      return () => clearTimeout(timer)
    }
  }, [gameState, timeRemaining])

  const handleStartGame = () => {
    if (wsRef.current && players.length >= 2 && !isStarting) {
      setIsStarting(true)
      wsRef.current.send(JSON.stringify({
        type: 'start_game',
        payload: {
          room_id: roomId,
          total_rounds: 10,
        },
      }))
    }
  }

  const handleGuess = (guessedPlayerId: string) => {
    if (wsRef.current && !hasGuessed) {
      wsRef.current.send(JSON.stringify({
        type: 'submit_guess',
        payload: {
          room_id: roomId,
          player_id: player.id,
          guessed_player_id: guessedPlayerId,
        },
      }))
      setHasGuessed(true)
    }
  }

  useEffect(() => {
    const audio = audioRef.current
    if (!audio) return

    const handleError = (e: Event) => {
      console.error('Audio error:', e)
      setAudioError('Failed to load audio preview')
    }

    audio.addEventListener('error', handleError)

    return () => {
      audio.removeEventListener('error', handleError)
    }
  }, [])

  const handleLeave = () => {
    if (wsRef.current) {
      wsRef.current.close()
    }
    onLeaveRoom()
  }

  const sortedPlayers = [...players].sort((a, b) => b.score - a.score)
  const isWinner = gameState === 'game_over' && sortedPlayers[0]?.id === player.id

  return (
      <>
    <audio
      ref={audioRef}
      crossOrigin="anonymous"
    />
    <div className="min-h-screen p-4 md:p-8">
      <div className="max-w-6xl mx-auto">
        {errorMessage && (
          <div className="bg-red-50 border-2 border-red-500 rounded-xl p-4 mb-6 flex items-center justify-between">
            <div className="flex items-center gap-3">
              <span className="text-2xl">⚠️</span>
              <div>
                <p className="font-bold text-red-900">Error</p>
                <p className="text-red-700">{errorMessage}</p>
              </div>
            </div>
            <button
              onClick={() => setErrorMessage(null)}
              className="text-red-500 hover:text-red-700 font-bold text-xl"
            >
              ✕
            </button>
          </div>
        )}
        
        <div className="bg-white rounded-2xl shadow-2xl p-6 mb-6 transition-all">
          <div className="flex justify-between items-center">
            <div>
              <h1 className="text-3xl font-bold text-gray-800">
                Room: {roomId}
              </h1>
              <p className="text-gray-600">Playing as: {player.name}</p>
            </div>
            <button
              onClick={handleLeave}
              disabled={isStarting}
              className="bg-red-500 hover:bg-red-600 disabled:bg-red-300 text-white font-bold py-2 px-6 rounded-lg transition-all"
            >
              Leave Room
            </button>
          </div>
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          <div className="lg:col-span-2 space-y-6">
            {gameState === 'waiting' && (
              <div className="bg-white rounded-2xl shadow-2xl p-8 transition-all">
                <h2 className="text-2xl font-bold text-gray-800 mb-4">
                  Waiting for Players...
                </h2>
                <p className="text-gray-600 mb-6">
                  {players.length} player{players.length !== 1 ? 's' : ''} in room
                </p>
                
                {players.length >= 2 ? (
                  <button
                    onClick={handleStartGame}
                    disabled={isStarting}
                    className={`w-full font-bold py-4 px-6 rounded-xl transition-all transform shadow-lg ${
                      isStarting 
                        ? 'bg-gray-400 cursor-not-allowed' 
                        : 'bg-green-500 hover:bg-green-600 hover:scale-105'
                    } text-white`}
                  >
                    {isStarting ? 'Starting...' : `Start Game (${players.length} players)`}
                  </button>
                ) : (
                  <div className="bg-yellow-50 border border-yellow-200 rounded-lg p-4">
                    <p className="text-yellow-800 text-center">
                      Need at least 2 players to start the game
                    </p>
                  </div>
                )}
              </div>
            )}

    {gameState === 'playing' && currentTrack && (
      <div className="bg-white rounded-2xl shadow-2xl p-8 transition-all">
        <div className="text-center mb-6">
          <div className="inline-block bg-purple-100 px-6 py-2 rounded-full">
            <span className="text-purple-800 font-bold">
              Round {currentRound} / {totalRounds}
            </span>
          </div>
          <div className="mt-4">
            <div className="text-4xl font-bold text-gray-800">
              {timeRemaining}s
            </div>
            <div className="w-full bg-gray-200 rounded-full h-2 mt-2">
              <div
                className="bg-purple-500 h-2 rounded-full transition-all duration-1000"
                style={{ width: `${(timeRemaining / 30) * 100}%` }}
              ></div>
            </div>
          </div>
        </div>

        {currentTrack.image_url && (
          <img
            src={currentTrack.image_url}
            alt={currentTrack.name}
            className="w-64 h-64 mx-auto rounded-xl shadow-lg mb-6"
          />
        )}

        <div className="text-center mb-6">
          <h3 className="text-2xl font-bold text-gray-800">
            {currentTrack.name}
          </h3>
          <p className="text-gray-600">{currentTrack.artists.join(', ')}</p>
        </div>

        {/* Simple Volume Control */}
        {currentTrack.preview_url ? (
          <div className="mb-6 flex items-center justify-center gap-3 bg-gray-100 rounded-lg p-4">
            <span className="text-gray-600 text-sm">🔊 Volume:</span>
            <input
              type="range"
              min="0"
              max="1"
              step="0.1"
              defaultValue="0.7"
              onChange={(e) => {
                if (audioRef.current) {
                  audioRef.current.volume = parseFloat(e.target.value)
                }
              }}
              className="w-32"
            />
          </div>
        ) : (
          <div className="bg-yellow-50 border border-yellow-200 rounded-lg p-4 mb-6">
            <p className="text-yellow-800 text-center">
              🔇 No preview available for this track
            </p>
          </div>
        )}

        {audioError && (
          <div className="bg-red-50 border border-red-200 rounded-lg p-4 mb-6">
            <p className="text-red-800 text-center text-sm">
              {audioError}
            </p>
          </div>
        )}

                <h4 className="text-xl font-semibold text-gray-800 mb-4 text-center">
                  Who listened to this the most?
                </h4>

                <div className="grid grid-cols-2 gap-4">
                  {players.map((p) => (
                    <button
                      key={p.id}
                      onClick={() => handleGuess(p.id)}
                      disabled={hasGuessed}
                      className={`py-4 px-6 rounded-lg font-bold transition-all transform ${
                        hasGuessed
                          ? 'bg-gray-300 cursor-not-allowed'
                          : 'bg-purple-500 hover:bg-purple-600 text-white hover:scale-105 shadow-lg'
                      }`}
                    >
                      {p.name}
                    </button>
                  ))}
                </div>

                <div className="mt-6 text-center text-gray-600">
                  {hasGuessed ? (
                    <span className="text-green-600 font-semibold">✓ Guess submitted!</span>
                  ) : (
                    <span>Make your guess...</span>
                  )}
                  <div className="text-sm mt-2">
                    {guessesCount} / {players.length} players have guessed
                  </div>
                </div>
              </div>
            )}

            {gameState === 'round_end' && roundResult && (
              <div className="bg-white rounded-2xl shadow-2xl p-8 transition-all">
                <h2 className="text-3xl font-bold text-center text-green-600 mb-6">
                  Round {roundResult.round} Complete!
                </h2>

                <div className="bg-green-50 border-2 border-green-200 rounded-xl p-6 mb-6">
                  <p className="text-center text-lg">
                    <strong>{players.find(p => p.id === roundResult.winner_id)?.name}</strong>{' '}
                    had this track ranked #{roundResult.winner_rank}!
                  </p>
                </div>

                {roundResult.correct_guessers && roundResult.correct_guessers.length > 0 && (
                  <div className="space-y-2">
                    <h3 className="font-bold text-gray-800">Correct Guesses:</h3>
                    {roundResult.correct_guessers.map((pid: string, idx: number) => (
                      <div key={pid} className="bg-blue-50 rounded-lg p-3">
                        <span className="font-semibold">
                          {players.find(p => p.id === pid)?.name}
                        </span>
                        <span className="text-blue-600 ml-2">
                          +{roundResult.points_awarded[pid]} points
                          {idx === 0 && ' ⚡ (Speed Bonus!)'}
                        </span>
                      </div>
                    ))}
                  </div>
                )}

                <div className="text-center mt-6 text-gray-600">
                  Next round starting soon...
                </div>
              </div>
            )}

            {gameState === 'game_over' && (
              <div className="bg-white rounded-2xl shadow-2xl p-8 transition-all">
                {isWinner ? (
                  <div className="text-center">
                    <div className="text-8xl mb-4 animate-bounce">🏆</div>
                    <h2 className="text-5xl font-bold bg-linear-to-r from-yellow-400 via-yellow-500 to-yellow-600 bg-clip-text text-transparent mb-4">
                      YOU WON!
                    </h2>
                    <p className="text-2xl text-gray-700 mb-8">
                      Congratulations, {player.name}!
                    </p>
                    <div className="bg-linear-to-r from-yellow-100 to-yellow-50 rounded-xl p-6 mb-8">
                      <p className="text-4xl font-bold text-yellow-700">
                        {sortedPlayers[0]?.score} points
                      </p>
                      <p className="text-gray-600 mt-2">Final Score</p>
                    </div>
                  </div>
                ) : (
                  <div className="text-center">
                    <div className="text-6xl mb-4">😔</div>
                    <h2 className="text-4xl font-bold text-gray-700 mb-4">
                      Game Over
                    </h2>
                    <p className="text-xl text-gray-600 mb-4">
                      Better luck next time, {player.name}!
                    </p>
                    <div className="bg-gray-100 rounded-xl p-6 mb-8">
                      <p className="text-3xl font-bold text-gray-700">
                        {players.find(p => p.id === player.id)?.score} points
                      </p>
                      <p className="text-gray-600 mt-2">
                        You finished #{sortedPlayers.findIndex(p => p.id === player.id) + 1}
                      </p>
                    </div>
                    <div className="bg-yellow-50 border border-yellow-200 rounded-xl p-4 mb-6">
                      <p className="font-semibold text-yellow-800 mb-2">👑 Winner:</p>
                      <p className="text-xl text-gray-800">
                        {sortedPlayers[0]?.name} - {sortedPlayers[0]?.score} points
                      </p>
                    </div>
                  </div>
                )}

                <div className="mt-8">
                  <h3 className="text-2xl font-bold text-center text-gray-800 mb-4">
                    Final Standings
                  </h3>
                  <div className="space-y-3">
                    {sortedPlayers.map((p, idx) => (
                      <div
                        key={p.id}
                        className={`flex justify-between items-center p-4 rounded-lg transition-all ${
                          idx === 0
                            ? 'bg-linear-to-r from-yellow-400 to-yellow-300 text-yellow-900 transform scale-105'
                            : p.id === player.id
                            ? 'bg-blue-100 border-2 border-blue-400'
                            : 'bg-gray-100'
                        }`}
                      >
                        <div className="flex items-center gap-3">
                          <span className="text-2xl font-bold">
                            {idx === 0 ? '👑' : `#${idx + 1}`}
                          </span>
                          <span className="font-semibold text-lg">
                            {p.name}
                            {p.id === player.id && ' (You)'}
                          </span>
                        </div>
                        <span className="text-2xl font-bold">{p.score} pts</span>
                      </div>
                    ))}
                  </div>
                </div>

                <button
                  onClick={handleLeave}
                  className="w-full mt-8 bg-purple-500 hover:bg-purple-600 text-white font-bold py-4 px-6 rounded-xl transition-all transform hover:scale-105"
                >
                  Back to Lobby
                </button>
              </div>
            )}
          </div>

          <div className="bg-white rounded-2xl shadow-2xl p-6 transition-all">
            <h3 className="text-xl font-bold text-gray-800 mb-4">Players</h3>
            <div className="space-y-3">
              {players.map((p) => (
                <div
                  key={p.id}
                  className={`flex justify-between items-center p-3 rounded-lg transition-all ${
                    p.id === player.id ? 'bg-purple-100 border-2 border-purple-300' : 'bg-gray-50'
                  }`}
                >
                  <span className="font-semibold">
                    {p.name}
                    {p.id === player.id && ' (You)'}
                  </span>
                  <span className="text-purple-600 font-bold">{p.score}</span>
                </div>
              ))}
            </div>
          </div>
        </div>
      </div>
    </div>
    </>
  )
}