import { afterEach, expect, it, vi } from "vitest";
import type { StagedUpload, UploadPage } from "./api";
import {
	completeDemoUpload,
	demoApi,
	SAMPLE_QUESTION,
	sampleAnswer,
} from "./demo-api";

it("provides an annotated example with references to two original sections", () => {
	const answer = sampleAnswer(SAMPLE_QUESTION);
	expect(answer.status).toBe("answered");
	expect(answer.response).toContain("[S1]");
	expect(answer.response).toContain("[S2]");
	expect(answer.sources.map((source) => source.section)).toEqual([
		"Design Requirements",
		"Solution",
	]);
});

afterEach(() => vi.unstubAllGlobals());

it("returns sample data without network requests", async () => {
	const fetcher = vi.fn();
	vi.stubGlobal("fetch", fetcher);
	const page = await demoApi<UploadPage>("/api/uploads?page=1");
	expect(page.uploads[0].originalName).toBe("portland-wharf.md");
	expect(
		sampleAnswer("What crane capacity is required?").sources[0].reference,
	).toBe("S1");
	expect(sampleAnswer("What is the manager's phone number?").status).toBe(
		"insufficient_evidence",
	);
	expect(fetcher).not.toHaveBeenCalled();
});

it("simulates upload metadata without transferring a file", async () => {
	const fetcher = vi.fn();
	vi.stubGlobal("fetch", fetcher);
	const result = await demoApi<StagedUpload>("/api/uploads", {
		method: "POST",
		body: JSON.stringify({ originalName: "preview.md", sizeBytes: 100 }),
	});
	expect(result.url.startsWith("demo://")).toBe(true);
	completeDemoUpload(result.upload.id);
	const page = await demoApi<UploadPage>("/api/uploads?page=1");
	expect(page.uploads[0].status).toBe("uploaded");
	expect(fetcher).not.toHaveBeenCalled();
});
