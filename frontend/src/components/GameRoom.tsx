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
  is_ready: boolean
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
  const [isReady, setIsReady] = useState(false)
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

    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const websocket = new WebSocket(`${protocol}//${window.location.host}/ws`)
    wsRef.current = websocket

    websocket.onopen = () => {
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

      switch (message.type) {
        case 'player_joined':
          setPlayers(message.payload.players || [])
          break

        case 'player_left':
          setPlayers(message.payload.players || [])
          break

        case 'player_ready':
          setPlayers(prev => prev.map(p => 
            p.id === message.payload.player_id 
              ? { ...p, is_ready: message.payload.is_ready }
              : p
          ))
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
            audioRef.current.play().catch(() => {
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
      // WebSocket disconnected
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

  const handleReady = () => {
    if (wsRef.current) {
      const newReadyState = !isReady
      setIsReady(newReadyState)
      wsRef.current.send(JSON.stringify({
        type: 'ready',
        payload: {
          is_ready: newReadyState,
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

    const handleError = () => {
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
          <div className="bg-red-500/10 border border-red-500/20 rounded-xl p-4 mb-6 flex items-center justify-between animate-shake">
            <div className="flex items-center gap-3">
              <span className="text-2xl">⚠️</span>
              <div>
                <p className="font-bold text-red-400">Error</p>
                <p className="text-red-200">{errorMessage}</p>
              </div>
            </div>
            <button
              onClick={() => setErrorMessage(null)}
              className="text-red-400 hover:text-red-200 font-bold text-xl"
            >
              ✕
            </button>
          </div>
        )}
        
        <div className="glass-panel rounded-2xl p-6 mb-6 transition-all flex justify-between items-center">
          <div>
            <h1 className="text-xl md:text-3xl font-bold text-white flex items-center gap-3">
              <span className="text-spotify-green">Room:</span> {roomId}
            </h1>
            <p className="text-gray-400">Playing as: <span className="text-white font-semibold">{player.name}</span></p>
          </div>
          <button
            onClick={handleLeave}
            disabled={isStarting}
            className="glass-button hover:bg-red-500/20 hover:border-red-500/50 text-red-400 hover:text-red-200 font-bold py-2 px-6 rounded-lg transition-all"
          >
            Leave Room
          </button>
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          <div className="lg:col-span-2 space-y-6">
            {gameState === 'waiting' && (
              <div className="glass-panel rounded-2xl p-8 md:p-12 text-center transition-all">
                <div className="text-5xl md:text-6xl mb-6 animate-bounce">🎵</div>
                <h2 className="text-2xl md:text-3xl font-bold text-white mb-4">
                  Waiting for Players...
                </h2>
                <p className="text-gray-400 mb-8 text-lg">
                  {players.length} player{players.length !== 1 ? 's' : ''} in room
                </p>
                
                <div className="flex flex-col items-center gap-4 mb-8">
                  <button
                    onClick={handleReady}
                    className={`px-8 py-3 rounded-full font-bold text-lg transition-all transform hover:scale-105 ${
                      isReady
                        ? 'bg-green-500/20 text-green-400 border-2 border-green-500'
                        : 'bg-gray-600 text-white hover:bg-gray-500'
                    }`}
                  >
                    {isReady ? '✓ Ready!' : 'Click to Ready Up'}
                  </button>
                </div>

                {players.length >= 2 ? (
                  <button
                    onClick={handleStartGame}
                    disabled={isStarting || !players.every(p => p.is_ready)}
                    className={`w-full font-bold py-4 px-6 rounded-full transition-all transform shadow-lg text-lg ${
                      isStarting || !players.every(p => p.is_ready)
                        ? 'bg-gray-600 cursor-not-allowed opacity-50' 
                        : 'bg-spotify-green hover:bg-[#1ed760] text-black hover:scale-105 hover:shadow-[0_0_20px_rgba(29,185,84,0.4)]'
                    }`}
                  >
                    {isStarting 
                      ? 'Starting Game...' 
                      : !players.every(p => p.is_ready)
                        ? 'Waiting for players to ready up...'
                        : `Start Game (${players.length} players)`}
                  </button>
                ) : (
                  <div className="bg-yellow-500/10 border border-yellow-500/20 rounded-xl p-4 inline-block">
                    <p className="text-yellow-200">
                      Need at least 2 players to start the game
                    </p>
                  </div>
                )}
              </div>
            )}

    {gameState === 'playing' && currentTrack && (
      <div className="glass-panel rounded-2xl p-8 transition-all relative overflow-hidden">
        {/* Background blur of album art */}
        {currentTrack.image_url && (
          <div 
            className="absolute inset-0 opacity-20 blur-3xl pointer-events-none"
            style={{ backgroundImage: `url(${currentTrack.image_url})`, backgroundSize: 'cover', backgroundPosition: 'center' }}
          />
        )}
        
        <div className="relative z-10">
          <div className="text-center mb-8">
            <div className="inline-block bg-white/10 backdrop-blur-md px-6 py-2 rounded-full border border-white/10 mb-6">
              <span className="text-spotify-green font-bold tracking-wider uppercase text-sm">
                Round {currentRound} / {totalRounds}
              </span>
            </div>
            
            <div className="flex items-center justify-center gap-4 mb-2">
              <div className="text-5xl font-bold text-white tabular-nums">
                {timeRemaining}s
              </div>
            </div>
            
            <div className="w-full bg-white/10 rounded-full h-2 mt-2 overflow-hidden">
              <div
                className="bg-spotify-green h-full rounded-full transition-all duration-1000 ease-linear shadow-[0_0_10px_rgba(29,185,84,0.5)]"
                style={{ width: `${(timeRemaining / 30) * 100}%` }}
              ></div>
            </div>
          </div>

          <div className="flex flex-col md:flex-row items-center gap-8 mb-8">
            {currentTrack.image_url ? (
              <div className="relative group">
                <img
                  src={currentTrack.image_url}
                  alt={currentTrack.name}
                  className="w-32 h-32 md:w-48 md:h-48 rounded-xl shadow-2xl group-hover:scale-105 transition-transform duration-500"
                />
                <div className="absolute inset-0 rounded-xl shadow-[inset_0_0_20px_rgba(0,0,0,0.5)] pointer-events-none"></div>
              </div>
            ) : (
              <div className="w-32 h-32 md:w-48 md:h-48 rounded-xl bg-gray-800 flex items-center justify-center shadow-2xl border border-white/10">
                <span className="text-4xl">❓</span>
              </div>
            )}

            <div className="text-center md:text-left flex-1">
              <h3 className="text-2xl md:text-3xl font-bold text-white mb-2 leading-tight">
                {currentTrack.name}
              </h3>
              <p className="text-lg md:text-xl text-gray-300">{currentTrack.artists.join(', ')}</p>
              
              {/* Simple Volume Control */}
              {currentTrack.preview_url ? (
                <div className="mt-6 hidden md:flex items-center justify-center md:justify-start gap-3">
                  <span className="text-gray-400 text-sm">Volume</span>
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
                    className="w-32 accent-spotify-green"
                  />
                </div>
              ) : (
                <div className="mt-4 text-yellow-400 text-sm flex items-center gap-2">
                  <span>🔇</span> No preview available
                </div>
              )}
            </div>
          </div>

          {audioError && (
            <div className="bg-red-500/10 border border-red-500/20 rounded-lg p-3 mb-6 text-center">
              <p className="text-red-300 text-sm">
                {audioError}
              </p>
            </div>
          )}

          <div className="border-t border-white/10 pt-8">
            <h4 className="text-xl font-semibold text-white mb-6 text-center">
              Who has this in their top tracks?
            </h4>

            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
              {players.map((p) => (
                <button
                  key={p.id}
                  onClick={() => handleGuess(p.id)}
                  disabled={hasGuessed}
                  className={`py-4 px-6 rounded-xl font-bold transition-all transform relative overflow-hidden group ${
                    hasGuessed
                      ? 'bg-gray-700/50 text-gray-500 cursor-not-allowed'
                      : 'glass-button hover:bg-spotify-green hover:text-black hover:border-spotify-green hover:scale-[1.02] hover:shadow-lg'
                  }`}
                >
                  <span className="relative z-10">{p.name}</span>
                </button>
              ))}
            </div>

            <div className="mt-6 text-center">
              {hasGuessed ? (
                <div className="inline-flex items-center gap-2 text-spotify-green font-bold bg-spotify-green/10 px-4 py-2 rounded-full">
                  <span>✓</span> Guess submitted!
                </div>
              ) : (
                <span className="text-gray-400 animate-pulse">Make your guess...</span>
              )}
              <div className="text-sm mt-3 text-gray-500">
                {guessesCount} / {players.length} players have guessed
              </div>
            </div>
          </div>
        </div>
      </div>
    )}

            {gameState === 'round_end' && roundResult && (
              <div className="glass-panel rounded-2xl p-8 transition-all text-center">
                <h2 className="text-4xl font-bold text-white mb-2">
                  Round Complete!
                </h2>
                <p className="text-gray-400 mb-8">Here's how it went down...</p>

                <div className="bg-linear-to-br from-purple-900/50 to-blue-900/50 border border-white/10 rounded-2xl p-8 mb-8 relative overflow-hidden">
                  <div className="absolute top-0 left-0 w-full h-1 bg-linear-to-r from-purple-500 to-blue-500"></div>
                  
                  {/* Revealed Track Info */}
                  <div className="flex flex-col items-center mb-6">
                    <img 
                      src={roundResult.track.image_url} 
                      alt={roundResult.track.name}
                      className="w-32 h-32 rounded-lg shadow-xl mb-4"
                    />
                    <h3 className="text-2xl font-bold text-white">{roundResult.track.name}</h3>
                    <p className="text-gray-300">{roundResult.track.artists.join(', ')}</p>
                  </div>

                  <div className="border-t border-white/10 pt-4">
                    <p className="text-lg text-gray-300 mb-2">The track belonged to...</p>
                    <p className="text-4xl font-bold text-white mb-2">
                      {players.find(p => p.id === roundResult.winner_id)?.name}
                    </p>
                    <div className="inline-block bg-white/10 px-4 py-1 rounded-full text-sm text-gray-300">
                      Ranked #{roundResult.winner_rank} in their top tracks
                    </div>
                  </div>
                </div>

                {roundResult.correct_guessers && roundResult.correct_guessers.length > 0 ? (
                  <div className="space-y-3 mb-8">
                    <h3 className="font-bold text-gray-300 uppercase tracking-wider text-sm mb-4">Correct Guesses</h3>
                    {roundResult.correct_guessers.map((pid: string, idx: number) => (
                      <div key={pid} className="bg-spotify-green/10 border border-spotify-green/20 rounded-xl p-4 flex justify-between items-center">
                        <span className="font-bold text-white">
                          {players.find(p => p.id === pid)?.name}
                        </span>
                        <span className="text-spotify-green font-bold flex items-center gap-2">
                          +{roundResult.points_awarded[pid]} pts
                          {idx === 0 && <span className="text-yellow-400 text-xs bg-yellow-400/10 px-2 py-1 rounded">⚡ FASTEST</span>}
                        </span>
                      </div>
                    ))}
                  </div>
                ) : (
                  <div className="bg-red-500/10 border border-red-500/20 rounded-xl p-6 mb-8">
                    <p className="text-red-300">No one guessed correctly! 😱</p>
                  </div>
                )}

                <div className="text-gray-500 animate-pulse">
                  Next round starting soon...
                </div>
              </div>
            )}

            {gameState === 'game_over' && (
              <div className="glass-panel rounded-2xl p-8 transition-all text-center">
                {isWinner ? (
                  <div className="mb-8">
                    <div className="text-6xl md:text-8xl mb-6 animate-bounce">🏆</div>
                    <h2 className="text-4xl md:text-6xl font-bold text-transparent bg-clip-text bg-linear-to-r from-yellow-300 via-yellow-500 to-yellow-600 mb-4">
                      VICTORY!
                    </h2>
                    <p className="text-xl md:text-2xl text-gray-300 mb-8">
                      You know your friends best, {player.name}!
                    </p>
                    <div className="bg-yellow-500/10 border border-yellow-500/30 rounded-2xl p-8 inline-block min-w-[200px]">
                      <p className="text-5xl font-bold text-yellow-400 mb-2">
                        {sortedPlayers[0]?.score}
                      </p>
                      <p className="text-yellow-200/70 uppercase tracking-wider text-sm">Final Score</p>
                    </div>
                  </div>
                ) : (
                  <div className="mb-8">
                    <div className="text-6xl mb-6">👏</div>
                    <h2 className="text-4xl font-bold text-white mb-4">
                      Game Over
                    </h2>
                    <p className="text-xl text-gray-400 mb-8">
                      Great game, {player.name}!
                    </p>
                    <div className="bg-white/5 rounded-2xl p-6 mb-8 inline-block min-w-[200px]">
                      <p className="text-4xl font-bold text-white mb-2">
                        {players.find(p => p.id === player.id)?.score}
                      </p>
                      <p className="text-gray-500 uppercase tracking-wider text-sm">Your Score</p>
                      <p className="text-gray-400 mt-2 text-sm">
                        Rank #{sortedPlayers.findIndex(p => p.id === player.id) + 1}
                      </p>
                    </div>
                  </div>
                )}

                <div className="max-w-md mx-auto">
                  <h3 className="text-xl font-bold text-gray-300 mb-6 uppercase tracking-wider">
                    Final Standings
                  </h3>
                  <div className="space-y-3">
                    {sortedPlayers.map((p, idx) => (
                      <div
                        key={p.id}
                        className={`flex justify-between items-center p-4 rounded-xl transition-all ${
                          idx === 0
                            ? 'bg-linear-to-r from-yellow-500/20 to-yellow-600/20 border border-yellow-500/50 transform scale-105 shadow-lg'
                            : p.id === player.id
                            ? 'bg-white/10 border border-white/30'
                            : 'bg-white/5 border border-white/5'
                        }`}
                      >
                        <div className="flex items-center gap-4">
                          <span className={`text-2xl font-bold w-8 ${idx === 0 ? 'text-yellow-400' : 'text-gray-500'}`}>
                            {idx === 0 ? '1' : idx + 1}
                          </span>
                          <div className="text-left">
                            <p className={`font-bold ${idx === 0 ? 'text-yellow-200' : 'text-white'}`}>
                              {p.name} {p.id === player.id && '(You)'}
                            </p>
                            {idx === 0 && <span className="text-xs text-yellow-500/80 uppercase font-bold">Winner</span>}
                          </div>
                        </div>
                        <span className="text-2xl font-bold text-white">{p.score}</span>
                      </div>
                    ))}
                  </div>
                </div>

                <button
                  onClick={handleLeave}
                  className="w-full mt-12 glass-button hover:bg-white/10 text-white font-bold py-4 px-6 rounded-xl transition-all"
                >
                  Back to Lobby
                </button>
              </div>
            )}
          </div>

          <div className="glass-panel rounded-2xl p-6 transition-all h-fit">
            <h3 className="text-xl font-bold text-white mb-6 flex items-center gap-2">
              <span>👥</span> Players
            </h3>
            <div className="space-y-3">
              {players.map((p) => (
                <div
                  key={p.id}
                  className={`flex justify-between items-center p-3 rounded-lg transition-all ${
                    p.id === player.id 
                      ? 'bg-spotify-green/10 border border-spotify-green/30' 
                      : 'bg-white/5 border border-white/5'
                  }`}
                >
                  <span className={`font-semibold ${p.id === player.id ? 'text-spotify-green' : 'text-gray-300'}`}>
                    {p.name}
                    {p.id === player.id && ' (You)'}
                  </span>
                  <div className="flex items-center gap-2">
                    {gameState === 'waiting' && (
                      <span className={`text-xs px-2 py-1 rounded-full ${
                        p.is_ready 
                          ? 'bg-green-500/20 text-green-400 border border-green-500/50' 
                          : 'bg-gray-700 text-gray-400'
                      }`}>
                        {p.is_ready ? 'READY' : 'NOT READY'}
                      </span>
                    )}
                    <span className="text-white font-bold bg-black/20 px-2 py-1 rounded text-sm">{p.score}</span>
                  </div>
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
