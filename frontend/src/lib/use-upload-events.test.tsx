import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, cleanup, renderHook } from "@testing-library/react";
import type { ReactNode } from "react";
import { afterEach, expect, it, vi } from "vitest";
import { useUploadEvents } from "./use-upload-events";

class FakeStream {
	static instances: FakeStream[] = [];
	onopen: (() => void) | null = null;
	onerror: (() => void) | null = null;
	change: (() => void) | null = null;
	close = vi.fn();
	constructor(public url: string) {
		FakeStream.instances.push(this);
	}
	addEventListener(_name: string, callback: () => void) {
		this.change = callback;
	}
}
afterEach(() => {
	cleanup();
	vi.useRealTimers();
	vi.unstubAllGlobals();
	FakeStream.instances = [];
});

it("coalesces progress events and reconciles after reconnecting", () => {
	vi.useFakeTimers();
	vi.stubGlobal("EventSource", FakeStream);
	const client = new QueryClient();
	const invalidate = vi.spyOn(client, "invalidateQueries").mockResolvedValue();
	const wrapper = ({ children }: { children: ReactNode }) => (
		<QueryClientProvider client={client}>{children}</QueryClientProvider>
	);
	const { result, unmount } = renderHook(() => useUploadEvents(true), {
		wrapper,
	});
	const first = FakeStream.instances[0];
	expect(first.url).toBe("/api/uploads/events");
	act(() => {
		first.onopen?.();
		first.change?.();
		first.change?.();
		vi.advanceTimersByTime(200);
	});
	expect(result.current).toBe(true);
	expect(invalidate).toHaveBeenCalledTimes(1);
	act(() => {
		first.onerror?.();
		vi.advanceTimersByTime(2000);
	});
	expect(first.close).toHaveBeenCalled();
	expect(result.current).toBe(false);
	expect(FakeStream.instances).toHaveLength(2);
	const second = FakeStream.instances[1];
	act(() => {
		second.onopen?.();
		vi.advanceTimersByTime(200);
	});
	expect(result.current).toBe(true);
	unmount();
	expect(second.close).toHaveBeenCalled();
	client.clear();
});
