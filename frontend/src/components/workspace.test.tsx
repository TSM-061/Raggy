import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import {
	cleanup,
	fireEvent,
	render,
	screen,
	waitFor,
} from "@testing-library/react";
import { afterEach, expect, it, vi } from "vitest";
import { WorkspaceApp } from "./workspace";

const clients: QueryClient[] = [];
afterEach(() => {
	cleanup();
	for (const client of clients) client.clear();
	clients.length = 0;
	vi.unstubAllGlobals();
});
const document = {
	id: "upload-1",
	originalName: "wharf-document.md",
	sizeBytes: 2048,
	status: "uploaded",
	createdAt: "2026-10-30T00:00:00Z",
};
const page = { uploads: [document], pageNumber: 1, pageSize: 20, total: 1 };
const answer = {
	status: "answered",
	response: "The deck supports 200-tonne cranes [S1].",
	sources: [
		{
			reference: "S1",
			chunk_id: "chunk-1",
			upload_id: "upload-1",
			chunk_index: 1,
			title: "Wharf document",
			section: "Design requirements",
			text: "Support cranes up to 200 tonnes.",
		},
	],
};

function mount() {
	const client = new QueryClient({
		defaultOptions: { queries: { retry: false } },
	});
	clients.push(client);
	render(
		<QueryClientProvider client={client}>
			<WorkspaceApp />
		</QueryClientProvider>,
	);
}

it("signs in after an expired session and loads real upload data", async () => {
	let signedIn = false;
	vi.stubGlobal(
		"fetch",
		vi.fn(async (path: string) => {
			if (path === "/api/auth/signin") {
				signedIn = true;
				return new Response(null, { status: 200 });
			}
			if (path === "/api/auth/refresh")
				return new Response(null, { status: 401 });
			return signedIn
				? Response.json(page)
				: new Response(null, { status: 401 });
		}),
	);
	mount();
	fireEvent.change(await screen.findByLabelText("Username"), {
		target: { value: "test-user" },
	});
	fireEvent.change(screen.getByLabelText("Password"), {
		target: { value: "test-password" },
	});
	fireEvent.click(
		screen.getByRole("button", { name: "Sign in to your workspace" }),
	);
	expect(await screen.findByText("wharf-document.md")).toBeTruthy();
});

it("shows a grounded answer and opens its original source", async () => {
	const fetcher = vi.fn(async (path: string) =>
		Response.json(path.startsWith("/api/search") ? answer : page),
	);
	vi.stubGlobal("fetch", fetcher);
	mount();
	const question = await screen.findByLabelText("What would you like to know?");
	fireEvent.change(question, {
		target: { value: "What can the deck support?" },
	});
	fireEvent.click(screen.getByRole("button", { name: "Ask question" }));
	expect(
		await screen.findByText(/The deck supports 200-tonne cranes/),
	).toBeTruthy();
	fireEvent.click(screen.getByRole("button", { name: "View source S1" }));
	expect(await screen.findByRole("dialog")).toBeTruthy();
	expect(screen.getByText("Support cranes up to 200 tonnes.")).toBeTruthy();
	expect(
		fetcher.mock.calls.some(([path]) =>
			path.includes("q=What+can+the+deck+support%3F"),
		),
	).toBe(true);
});

it("keeps insufficient evidence distinct from request failures", async () => {
	vi.stubGlobal(
		"fetch",
		vi.fn(async (path: string) =>
			Response.json(
				path.startsWith("/api/search")
					? {
							status: "insufficient_evidence",
							response: "The documents do not cover this topic.",
							sources: [],
						}
					: page,
			),
		),
	);
	mount();
	fireEvent.change(
		await screen.findByLabelText("What would you like to know?"),
		{ target: { value: "Unsupported question" } },
	);
	fireEvent.click(screen.getByRole("button", { name: "Ask question" }));
	expect(await screen.findByText("More information needed")).toBeTruthy();
	expect(screen.queryByRole("alert")).toBeNull();
});

it("retains a failed question so it can be retried", async () => {
	vi.stubGlobal(
		"fetch",
		vi.fn(async (path: string) =>
			path.startsWith("/api/search")
				? new Response("internal", { status: 500 })
				: Response.json(page),
		),
	);
	mount();
	const field = await screen.findByLabelText("What would you like to know?");
	fireEvent.change(field, { target: { value: "Keep my question" } });
	fireEvent.click(screen.getByRole("button", { name: "Ask question" }));
	expect(await screen.findByRole("alert")).toBeTruthy();
	expect((field as HTMLTextAreaElement).value).toBe("Keep my question");
});

it("validates file types and completes a signed upload", async () => {
	const fetcher = vi.fn(async (path: string, options?: RequestInit) => {
		if (path === "https://storage.example/upload")
			return new Response(null, { status: 200 });
		if (options?.method === "POST")
			return Response.json({
				upload: document,
				url: "https://storage.example/upload",
				signedHeaders: { "x-amz-meta-example": "signed" },
			});
		return Response.json(page);
	});
	vi.stubGlobal("fetch", fetcher);
	mount();
	await screen.findByText("wharf-document.md");
	fireEvent.click(screen.getByRole("button", { name: "Upload document" }));
	const input = screen.getByLabelText("Choose Markdown document");
	fireEvent.change(input, {
		target: { files: [new File(["bad"], "document.pdf")] },
	});
	expect(await screen.findByRole("alert")).toBeTruthy();
	fireEvent.change(input, {
		target: { files: [new File(["# Document"], "new-document.md")] },
	});
	fireEvent.click(screen.getByRole("button", { name: "Upload document" }));
	expect(await screen.findByText("Your document is on its way")).toBeTruthy();
	expect(
		fetcher.mock.calls.find(
			([path]) => path === "https://storage.example/upload",
		)?.[1]?.credentials,
	).toBe("omit");
});

it("shows an actionable empty state and disables questions without documents", async () => {
	vi.stubGlobal(
		"fetch",
		vi.fn(async () => Response.json({ ...page, uploads: [], total: 0 })),
	);
	mount();
	expect(
		await screen.findByText("Your next answer starts with a document."),
	).toBeTruthy();
	expect(
		(screen.getByRole("button", { name: "Ask question" }) as HTMLButtonElement)
			.disabled,
	).toBe(true);
	fireEvent.click(screen.getAllByRole("button", { name: /Documents/ })[0]);
	expect(
		await screen.findByText("Make room for your first document."),
	).toBeTruthy();
});

it("shows a connection error instead of pretending the workspace is empty", async () => {
	vi.stubGlobal(
		"fetch",
		vi.fn().mockRejectedValue(new TypeError("Network error")),
	);
	mount();
	expect(
		await screen.findByText("We couldn’t reach your workspace"),
	).toBeTruthy();
	expect(screen.getByRole("button", { name: "Try again" })).toBeTruthy();
});

it("signs out and removes workspace data", async () => {
	vi.stubGlobal(
		"fetch",
		vi.fn(async (path: string) =>
			path === "/api/auth/signout"
				? new Response(null, { status: 200 })
				: Response.json(page),
		),
	);
	mount();
	await screen.findByText("wharf-document.md");
	fireEvent.click(screen.getAllByRole("button", { name: "Sign out" })[0]);
	await waitFor(() => expect(screen.getByLabelText("Username")).toBeTruthy());
	expect(screen.queryByText("wharf-document.md")).toBeNull();
});

it("retries a failed file transfer without creating a second upload record", async () => {
	let puts = 0;
	let creates = 0;
	vi.stubGlobal(
		"fetch",
		vi.fn(async (path: string, options?: RequestInit) => {
			if (path === "https://storage.example/retry") {
				puts++;
				return new Response(null, { status: puts === 1 ? 500 : 200 });
			}
			if (options?.method === "POST") {
				creates++;
				return Response.json({
					upload: document,
					url: "https://storage.example/retry",
					signedHeaders: {},
				});
			}
			return Response.json(page);
		}),
	);
	mount();
	await screen.findByText("wharf-document.md");
	fireEvent.click(screen.getByRole("button", { name: "Upload document" }));
	fireEvent.change(screen.getByLabelText("Choose Markdown document"), {
		target: { files: [new File(["# Document"], "document.md")] },
	});
	fireEvent.click(screen.getByRole("button", { name: "Upload document" }));
	fireEvent.click(await screen.findByRole("button", { name: "Retry upload" }));
	expect(await screen.findByText("Your document is on its way")).toBeTruthy();
	expect(creates).toBe(1);
	expect(puts).toBe(2);
});

it("fetches the next document page", async () => {
	vi.stubGlobal(
		"fetch",
		vi.fn(async (path: string) =>
			Response.json(
				path.includes("page=2&")
					? {
							...page,
							uploads: [
								{ ...document, id: "next", originalName: "next-page.md" },
							],
							pageNumber: 2,
							total: 21,
						}
					: { ...page, total: 21 },
			),
		),
	);
	mount();
	await screen.findByText("wharf-document.md");
	fireEvent.click(screen.getAllByRole("button", { name: /Documents/ })[0]);
	fireEvent.click(screen.getByRole("button", { name: "Next page" }));
	expect(await screen.findByText("next-page.md")).toBeTruthy();
	expect(screen.getByText("Page 2 of 2")).toBeTruthy();
});

it("cancels a pending question without presenting it as a failure", async () => {
	let aborted = false;
	vi.stubGlobal(
		"fetch",
		vi.fn(async (path: string, options?: RequestInit) => {
			if (!path.startsWith("/api/search")) return Response.json(page);
			return new Promise<Response>((_resolve, reject) => {
				options?.signal?.addEventListener("abort", () => {
					aborted = true;
					reject(new DOMException("Aborted", "AbortError"));
				});
			});
		}),
	);
	mount();
	fireEvent.change(
		await screen.findByLabelText("What would you like to know?"),
		{ target: { value: "Cancel this question" } },
	);
	fireEvent.click(screen.getByRole("button", { name: "Ask question" }));
	fireEvent.click(await screen.findByRole("button", { name: "Cancel" }));
	expect(aborted).toBe(true);
	expect(screen.queryByRole("alert")).toBeNull();
	expect(
		(screen.getByRole("button", { name: "Ask question" }) as HTMLButtonElement)
			.disabled,
	).toBe(false);
});
