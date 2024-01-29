// ConnectRPC client configuration for Duck API
import { createClient } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import { DuckAPI } from "@/gen/api_connect";

// Create the transport
const transport = createConnectTransport({
  baseUrl: import.meta.env.VITE_API_SERVER || "http://localhost:1323",
});

// Create the ConnectRPC client
export const duckApiClient = createClient(DuckAPI, transport);

// Export for direct use in components/stores
export default duckApiClient;
