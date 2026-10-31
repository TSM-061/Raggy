import { useQueryClient } from "@tanstack/react-query";
import { useEffect, useState } from "react";
import { DEMO_MODE } from "./demo-api";

/** Keeps document queries in sync with the server’s progress notifications. */
export function useUploadEvents(enabled: boolean) {
	const client = useQueryClient();
	const [connected, setConnected] = useState(false);
	useEffect(() => {
		if (!enabled || DEMO_MODE || typeof EventSource === "undefined") return;
		let stopped = false;
		let stream: EventSource | undefined;
		let reconnect: ReturnType<typeof setTimeout> | undefined;
		let refresh: ReturnType<typeof setTimeout> | undefined;
		let attempts = 0;
		function invalidate() {
			if (refresh) return;
			refresh = setTimeout(() => {
				refresh = undefined;
				void client.invalidateQueries(
					{ queryKey: ["uploads"] },
					{ cancelRefetch: false },
				);
			}, 200);
		}
		function open() {
			if (stopped) return;
			stream = new EventSource("/api/uploads/events");
			stream.onopen = () => {
				setConnected(true);
				attempts = 0;
				invalidate();
			};
			stream.addEventListener("uploads.changed", invalidate);
			stream.onerror = () => {
				setConnected(false);
				stream?.close();
				invalidate(); // REST handles cookie refresh before reconnecting the stream.
				reconnect = setTimeout(
					open,
					Math.min(30000, 2000 * 2 ** Math.min(attempts++, 4)),
				);
			};
		}
		open();
		return () => {
			stopped = true;
			stream?.close();
			clearTimeout(reconnect);
			clearTimeout(refresh);
			setConnected(false);
		};
	}, [client, enabled]);
	return connected;
}
