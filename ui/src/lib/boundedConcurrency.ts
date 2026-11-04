/**
 * Upper bound on parallel writes during a .env import. Four requests in flight
 * turns queue latency from O(n) round-trips into O(n / 4) while keeping the
 * single SQLite writer from being flooded (issue #309).
 */
export const IMPORT_CONCURRENCY_LIMIT = 4

/**
 * Runs `task` over `items` with at most `limit` promises in flight, returning
 * results in input order regardless of completion order. Rejections are
 * propagated as soon as a worker throws; callers that need per-item errors
 * should catch inside `task`.
 */
export async function mapWithConcurrency<T, R>(
  items: readonly T[],
  limit: number,
  task: (item: T, index: number) => Promise<R>,
): Promise<R[]> {
  if (items.length === 0) return []
  const results = new Array<R>(items.length)
  const workerCount = Math.max(1, Math.min(limit, items.length))
  let next = 0
  let aborted = false

  const worker = async (): Promise<void> => {
    while (!aborted && next < items.length) {
      const index = next
      next += 1
      try {
        results[index] = await task(items[index], index)
      } catch (err) {
        aborted = true
        throw err
      }
    }
  }

  await Promise.all(Array.from({ length: workerCount }, worker))
  return results
}
