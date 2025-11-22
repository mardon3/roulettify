import { createRoot } from 'react-dom/client'
import './index.css'
import App from './App.tsx'

// Strict Mode disabled to prevent double WebSocket connections
// React 18's Strict Mode intentionally double-mounts components in development
// This causes WebSocket connections to open twice, leading to connection issues
createRoot(document.getElementById('root')!).render(
  <App />
)