import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { BrowserRouter } from 'react-router-dom'
import '@uiw/react-md-editor/markdown-editor.css'
import './index.css'
import App from './App.tsx'

// react-md-editor reads data-color-mode from body (not html).
document.body.setAttribute('data-color-mode', 'dark')

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <BrowserRouter>
      <App />
    </BrowserRouter>
  </StrictMode>,
)
