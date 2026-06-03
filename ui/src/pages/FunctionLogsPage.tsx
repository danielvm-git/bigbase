import { useParams, Navigate } from 'react-router-dom'

/** @deprecated Route redirects to FunctionDetailPage logs tab; kept for test imports. */
export default function FunctionLogsPage() {
  const { id } = useParams<{ id: string }>()
  if (!id) return null
  return <Navigate to={`/functions/${id}?tab=logs`} replace />
}
