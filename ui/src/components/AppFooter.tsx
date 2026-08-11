interface AppFooterProps {
  appVersion?: string | null
  showTutorial?: boolean
  onOpenTutorial?: () => void
}

export function AppFooter({ appVersion, showTutorial = false, onOpenTutorial }: AppFooterProps) {
  return (
    <footer className="app-footer" data-testid="app-footer">
      <div className="app-footer-brand">
        <span className="app-footer-logo" aria-hidden>B</span>
        <span>© 2026 BigBase · MIT License</span>
      </div>
      <div className="app-footer-meta">
        {showTutorial && (
          <button
            type="button"
            className="btn btn-secondary btn-sm"
            onClick={onOpenTutorial}
            data-testid="tutorial-btn"
          >
            Help
          </button>
        )}
        {appVersion && <span className="mono">v{appVersion}</span>}
        <span className="app-footer-sep" aria-hidden />
        <span>
          Built with{' '}
          <a href="https://github.com/danielvm-git/bigpowers" target="_blank" rel="noopener noreferrer">
            BigPowers
          </a>
          {' '}by{' '}
          <a href="https://github.com/danielvm-git" target="_blank" rel="noopener noreferrer">
            danielvm-git
          </a>
        </span>
        <span className="app-footer-sep" aria-hidden />
        <a href="https://github.com/danielvm-git/bigbase" target="_blank" rel="noopener noreferrer">
          GitHub
        </a>
        <a
          href="https://github.com/danielvm-git/bigbase/blob/main/CHANGELOG.md"
          target="_blank"
          rel="noopener noreferrer"
        >
          Changelog
        </a>
        <a
          href="https://github.com/danielvm-git/bigbase/blob/main/specs/tech-architecture/GLOSSARY_LATEST.md"
          target="_blank"
          rel="noopener noreferrer"
        >
          Glossary
        </a>
      </div>
    </footer>
  )
}
