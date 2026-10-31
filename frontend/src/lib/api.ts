export type Upload = {
	id: string;
	originalName: string;
	sizeBytes: number;
	status: string;
	chunkTotal?: number | null;
	chunksProcessed?: number;
	createdAt: string;
};
export type UploadPage = {
	uploads: Upload[];
	pageNumber: number;
	pageSize: number;
	total: number;
};
export type Source = {
	reference: string;
	chunk_id: string;
	upload_id: string;
	chunk_index: number;
	title?: string;
	section?: string;
	text: string;
};
export type Answer = {
	status: "answered" | "insufficient_evidence";
	response: string;
	sources: Source[];
};
export type StagedUpload = {
	upload: Upload;
	url: string;
	signedHeaders: Record<string, string>;
};

export class SessionExpired extends Error {
	constructor() {
		super("Your session has ended. Sign in to continue.");
	}
}

let refreshRequest: Promise<void> | undefined;
/** Renews the browser session while sharing concurrent refresh requests. */
async function refreshSession() {
	if (!refreshRequest) {
		refreshRequest = fetch("/api/auth/refresh", {
			method: "POST",
			credentials: "same-origin",
		})
			.then((res) => {
				if ([400, 401, 403].includes(res.status)) throw new SessionExpired();
				if (!res.ok) throw new Error("Could not reconnect. Please try again.");
			})
			.finally(() => {
				refreshRequest = undefined;
			});
	}
	return refreshRequest;
}

/** Calls the application API and refreshes an expired session before retrying. */
export async function api<T>(path: string, init?: RequestInit): Promise<T> {
	if (DEMO_MODE) return demoApi<T>(path, init);
	const options = { ...init, credentials: "same-origin" as const };
	let res = await fetch(path, options);
	if (res.status === 401 && !path.startsWith("/api/auth/")) {
		await refreshSession();
		res = await fetch(path, options);
		if (res.status === 401) throw new SessionExpired();
	}
	if (!res.ok) {
		if (path === "/api/auth/signin" && [401, 400].includes(res.status))
			throw new Error("Check your username and password and try again.");
		throw new Error(
			res.status >= 500
				? "The service couldn’t complete that request. Please try again."
				: "That request couldn’t be completed. Please try again.",
		);
	}
	if (res.status === 204 || path.startsWith("/api/auth/"))
		return undefined as T;
	return res.json() as Promise<T>;
}

export function listUploads(page: number, signal?: AbortSignal) {
	return api<UploadPage>(`/api/uploads?page=${page}&pageSize=20`, { signal });
}

/** Creates document metadata and requests signed storage-upload details. */
export function stageUpload(file: File) {
	return api<StagedUpload>("/api/uploads", {
		method: "POST",
		headers: { "Content-Type": "application/json" },
		body: JSON.stringify({
			originalName: file.name,
			contentType: "text/markdown",
			profileHint: "profile-hint.project-report-md",
			sizeBytes: file.size,
		}),
	});
}

/** Transfers a selected file using its signed upload authorization. */
export async function putUpload(staged: StagedUpload, file: File) {
	if (DEMO_MODE) {
		completeDemoUpload(staged.upload.id);
		return;
	}
	// Storage receives only signed headers, never session cookies.
	const res = await fetch(staged.url, {
		method: "PUT",
		headers: staged.signedHeaders,
		body: file,
		credentials: "omit",
	});
	if (!res.ok)
		throw new Error(
			"Your file couldn’t finish uploading. Try again, or choose the file again to start a fresh upload.",
		);
}

/** Queries the workspace for an answer and its supporting sources. */
export function askQuestion(question: string, signal: AbortSignal) {
	return api<Answer>(
		`/api/search?${new URLSearchParams({ q: question, limit: "5" })}`,
		{ signal },
	);
}

export function formatBytes(bytes: number) {
	if (bytes < 1024) return `${bytes} B`;
	return bytes < 1024 * 1024
		? `${Math.round(bytes / 1024)} KB`
		: `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

import { completeDemoUpload, DEMO_MODE, demoApi } from "./demo-api";
