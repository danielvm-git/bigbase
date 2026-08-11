import { useContext } from 'react'
import { AuthContext } from '../context/authState'

/** Access-token session state: remaining time, refresh, pending route. */
export function useAuthSession() {
  return useContext(AuthContext)
}
