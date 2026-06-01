interface PreviewBannerProps {
  message?: string
}

export function PreviewBanner({ message }: PreviewBannerProps) {
  return (
    <div className="preview-banner" role="status">
      <span className="preview-banner-label">Preview</span>
      <span>{message ?? 'Using sample data or partial backend — GitHub and deploy APIs may be simulated.'}</span>
    </div>
  )
}
