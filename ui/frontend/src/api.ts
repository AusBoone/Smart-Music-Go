/**
 * Utility for fetching JSON from the backend API with consistent error handling.
 *
 * @param path - API path beginning with '/api/'.
 * @param options - optional fetch configuration.
 * @returns parsed JSON of type T.
 * @throws Error when the response status is not OK.
 */
export async function api<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await fetch(path, options);
  if (!res.ok) {
    let msg = 'Request failed';
    try {
      const data = await res.json();
      msg = data.error || msg;
    } catch {
      // ignore JSON parse errors
    }
    throw new Error(msg);
  }
  return res.json();
}
