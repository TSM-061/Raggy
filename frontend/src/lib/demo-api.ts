import type { Answer, Source, StagedUpload, Upload } from "./api";

export const DEMO_MODE = import.meta.env.VITE_DEMO_MODE === "true";
export const SAMPLE_QUESTION =
	"What are the wharf's key requirements and proposed construction solution?";

const documents: Upload[] = [
	{
		id: "demo-wharf",
		originalName: "portland-wharf.md",
		sizeBytes: 1850,
		status: "processed",
		chunkTotal: 4,
		chunksProcessed: 4,
		createdAt: "2026-10-30T00:00:00Z",
	},
	{
		id: "demo-terminal",
		originalName: "auckland-terminal.md",
		sizeBytes: 1860,
		status: "processing",
		chunkTotal: 6,
		chunksProcessed: 2,
		createdAt: "2026-10-29T00:00:00Z",
	},
	{
		id: "demo-jetty",
		originalName: "palau-jetty.md",
		sizeBytes: 1827,
		status: "pending",
		createdAt: "2026-10-29T00:00:00Z",
	},
];

const source: Source = {
	reference: "S1",
	chunk_id: "demo-wharf-requirements",
	upload_id: "demo-wharf",
	chunk_index: 1,
	title: "Heavy-Lift Material Wharf Expansion",
	section: "Design Requirements",
	text: "- Support heavy track-mounted mobile cranes (up to 200-tonne operating weight) on the wharf deck.\n- Interface with bulk material barges (90m in length and up to 5,000 deadweight tonnage).\n- Maintain structural integrity under severe cyclone conditions (Category 5 wind and wave loading).\n- Utilize a pre-cast concrete methodology to minimize over-water formwork and concrete pouring.",
};

/** Selects a canned answer and source references for the interface preview. */
export function sampleAnswer(question: string): Answer {
	const q = question.toLowerCase();
	if (q === SAMPLE_QUESTION.toLowerCase()) {
		return {
			status: "answered",
			response:
				"The wharf must support mobile cranes up to 200 tonnes and accommodate 90-metre barges of up to 5,000 deadweight tonnage [S1]. It must also withstand Category 5 cyclone conditions [S1].\n\nThe proposed solution is a suspended pre-cast concrete deck on raked, heavy-wall steel tubular piles. Pre-cast headstocks and plank units allow construction to proceed top-down from the shore [S2].\n\nThe design also includes an impressed current cathodic protection system to address marine corrosion [S2].",
			sources: [
				source,
				{
					...source,
					reference: "S2",
					chunk_id: "demo-wharf-solution",
					chunk_index: 3,
					section: "Solution",
					text: "The resulting infrastructure consists of a high-strength, suspended pre-cast concrete deck supported by a grid of raked, heavy-wall steel tubular piles. The design optimized the placement of pre-cast headstocks and plank units, allowing the contractor to build out top-down from the shore without relying on specialized jack-up rigs.\n\nThe structure features integrated cast-in-situ crane rail recesses, quick-release mooring bollards, and an impressed current cathodic protection (ICCP) system to combat aggressive tropical marine corrosion.",
				},
			],
		};
	}
	if (/crane|weight|requirement|wharf|design|summari[sz]e|overview/.test(q)) {
		return {
			status: "answered",
			response:
				"The wharf document calls for a deck that supports mobile cranes up to 200 tonnes and accommodates 90-metre barges of up to 5,000 deadweight tonnage [S1]. It also specifies resistance to Category 5 cyclone conditions and a pre-cast concrete construction approach [S1].",
			sources: [source],
		};
	}
	if (/risk|cyclone/.test(q)) {
		return {
			status: "answered",
			response:
				"The sample wharf requirements highlight severe cyclone conditions: the structure must maintain integrity under Category 5 wind and wave loading [S1].",
			sources: [source],
		};
	}
	return {
		status: "insufficient_evidence",
		response:
			"This preview has no sample answer for that question. Try asking about the wharf’s crane capacity, design requirements, or cyclone risks.",
		sources: [],
	};
}

// This adapter is opt-in and never contacts the backend or an AI provider.
// Its records live only in memory and reset on a browser reload.
export async function demoApi<T>(path: string, init?: RequestInit): Promise<T> {
	const url = new URL(path, "https://preview.invalid");
	if (init?.signal?.aborted) throw new DOMException("Aborted", "AbortError");
	if (url.pathname.startsWith("/api/auth/")) return undefined as T;
	if (url.pathname === "/api/search")
		return sampleAnswer(url.searchParams.get("q") || "") as T;
	if (url.pathname === "/api/uploads" && init?.method === "POST") {
		const body = JSON.parse(String(init.body)) as {
			originalName: string;
			sizeBytes: number;
		};
		const record: Upload = {
			id: `demo-${crypto.randomUUID()}`,
			originalName: body.originalName,
			sizeBytes: body.sizeBytes,
			status: "pending",
			createdAt: new Date().toISOString(),
		};
		documents.unshift(record);
		return {
			upload: record,
			url: `demo://upload/${record.id}`,
			signedHeaders: {},
		} as StagedUpload as T;
	}
	if (url.pathname === "/api/uploads") {
		const page = Math.max(1, Number(url.searchParams.get("page")) || 1);
		return {
			uploads: documents.slice((page - 1) * 20, page * 20),
			pageNumber: page,
			pageSize: 20,
			total: documents.length,
		} as T;
	}
	throw new Error("This action is not available in the sample preview.");
}

/** Marks a simulated upload complete in the preview’s document list. */
export function completeDemoUpload(id: string) {
	const document = documents.find((item) => item.id === id);
	if (document) document.status = "uploaded";
}
