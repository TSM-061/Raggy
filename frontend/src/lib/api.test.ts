import { afterEach, describe, expect, it, vi } from "vitest";
import { api, putUpload, SessionExpired, stageUpload } from "./api";

afterEach(() => vi.unstubAllGlobals());

describe("session requests", () => {
	it("refreshes an expired access token and retries once", async () => {
		const fetcher = vi
			.fn()
			.mockResolvedValueOnce(new Response(null, { status: 401 }))
			.mockResolvedValueOnce(new Response(null, { status: 200 }))
			.mockResolvedValueOnce(Response.json({ uploads: [] }));
		vi.stubGlobal("fetch", fetcher);
		await expect(api("/api/uploads")).resolves.toEqual({ uploads: [] });
		expect(fetcher.mock.calls.map(([path]) => path)).toEqual([
			"/api/uploads",
			"/api/auth/refresh",
			"/api/uploads",
		]);
		expect(fetcher.mock.calls[0][1].credentials).toBe("same-origin");
	});

	it("returns to sign-in when refresh is rejected", async () => {
		vi.stubGlobal(
			"fetch",
			vi
				.fn()
				.mockResolvedValueOnce(new Response(null, { status: 401 }))
				.mockResolvedValueOnce(new Response(null, { status: 400 })),
		);
		await expect(api("/api/uploads")).rejects.toBeInstanceOf(SessionExpired);
	});

	it("never loops when the retried request is unauthorized", async () => {
		const fetcher = vi
			.fn()
			.mockResolvedValueOnce(new Response(null, { status: 401 }))
			.mockResolvedValueOnce(new Response(null, { status: 200 }))
			.mockResolvedValueOnce(new Response(null, { status: 401 }));
		vi.stubGlobal("fetch", fetcher);
		await expect(api("/api/uploads")).rejects.toBeInstanceOf(SessionExpired);
		expect(fetcher).toHaveBeenCalledTimes(3);
	});

	it("does not expose internal provider errors", async () => {
		vi.stubGlobal(
			"fetch",
			vi
				.fn()
				.mockResolvedValue(
					new Response("private provider output", { status: 500 }),
				),
		);
		await expect(api("/api/search?q=test")).rejects.toThrow(
			"The service couldn’t complete that request.",
		);
	});
});

it("creates Markdown upload metadata and sends only signed headers to storage", async () => {
	const staged = {
		upload: { id: "upload-1" },
		url: "https://storage.example/upload?signature=test",
		signedHeaders: { "x-amz-meta-traceparent": "trace" },
	};
	const fetcher = vi
		.fn()
		.mockResolvedValueOnce(Response.json(staged))
		.mockResolvedValueOnce(new Response(null, { status: 200 }));
	vi.stubGlobal("fetch", fetcher);
	const file = new File(["# Report"], "report.md", { type: "text/markdown" });
	const result = await stageUpload(file);
	await putUpload(result, file);
	expect(JSON.parse(fetcher.mock.calls[0][1].body)).toEqual({
		originalName: "report.md",
		contentType: "text/markdown",
		profileHint: "profile-hint.project-report-md",
		sizeBytes: file.size,
	});
	expect(fetcher.mock.calls[1]).toEqual([
		staged.url,
		{
			method: "PUT",
			headers: staged.signedHeaders,
			body: file,
			credentials: "omit",
		},
	]);
});
