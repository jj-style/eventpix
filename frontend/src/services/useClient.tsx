import { Client, createClient } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import { useMemo } from "react";
import { ServiceType } from "@bufbuild/protobuf";


// This transport is going to be used throughout the app
const transport = createConnectTransport({
  baseUrl: "http://localhost:8080",
});

/**
* Get a promise client for the given service.
*/
export function useClient<T extends ServiceType>(service: T): Client<T> {
  // We memoize the client, so that we only create one instance per service.
  return useMemo(() => createClient(service, transport), [service]);
}