import { QueryClient } from '@tanstack/react-query'

/** Creates the query cache exposed through the router context. */
export function getContext() {
  const queryClient = new QueryClient()

  return {
    queryClient,
  }
}
export default function TanstackQueryProvider() {}
