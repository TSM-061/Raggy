import {
	ArrowUpRight,
	BookOpen,
	Check,
	Copy,
	FileText,
	SearchX,
} from "lucide-react";
import { Fragment, useEffect, useState } from "react";
import { Badge } from "#/components/ui/badge";
import { Button } from "#/components/ui/button";
import { Card, CardContent } from "#/components/ui/card";
import {
	Dialog,
	DialogContent,
	DialogDescription,
	DialogHeader,
	DialogTitle,
} from "#/components/ui/dialog";
import type { Answer, Source } from "#/lib/api";
import { DEMO_MODE } from "#/lib/demo-api";

/** Displays an answer and lets readers inspect its cited source text. */
export function AnswerPanel({
	answer,
	question,
}: {
	answer: Answer;
	question: string;
}) {
	const [source, setSource] = useState<Source | null>(null);
	const [copied, setCopied] = useState(false);
	const [copyError, setCopyError] = useState(false);
	useEffect(() => {
		if (!copied) return;
		const timer = window.setTimeout(() => setCopied(false), 2000);
		return () => window.clearTimeout(timer);
	}, [copied]);

	async function copy() {
		try {
			await navigator.clipboard.writeText(answer.response);
			setCopied(true);
			setCopyError(false);
		} catch {
			setCopyError(true);
		}
	}

	const insufficient = answer.status === "insufficient_evidence";
	let offset = 0;
	const parts = answer.response.split(/(\[S\d+\])/g).map((part) => {
		const token = { part, offset };
		offset += part.length;
		return token;
	});
	const references = new Map(
		answer.sources.map((item) => [item.reference, item]),
	);
	return (
		<section
			aria-label="Answer"
			className="space-y-5 animate-in fade-in duration-300"
		>
			<div className="flex items-start gap-3">
				<span className="mt-1 h-5 w-0.5 rounded bg-primary/40" />
				<h2 className="text-base font-medium leading-7">{question}</h2>
			</div>
			<Card className="overflow-hidden border-primary/15 shadow-sm">
				<CardContent className="p-5 sm:p-7">
					<div className="mb-5 flex items-center justify-between gap-3">
						<span className="flex items-center gap-2 text-sm font-semibold">
							<span className="flex size-7 items-center justify-center rounded-lg bg-primary/10 text-primary">
								{insufficient ? (
									<SearchX className="size-4" />
								) : (
									<BookOpen className="size-4" />
								)}
							</span>
							{insufficient
								? "More information needed"
								: DEMO_MODE
									? "Sample answer"
									: "Workspace answer"}
						</span>
						<Badge variant="outline" className="text-[10px] font-medium">
							{insufficient ? "Insufficient evidence" : "With sources"}
						</Badge>
					</div>
					<div className="whitespace-pre-wrap text-[15px] leading-8 text-foreground/85">
						{parts.map(({ part, offset }) => {
							const match = references.get(part.slice(1, -1));
							// Parts are positional tokens in a single immutable answer.
							return match ? (
								<button
									key={`${offset}-${part}`}
									type="button"
									onClick={() => setSource(match)}
									aria-label={`View source ${match.reference}`}
									className="relative -top-0.5 mr-[0.25em] inline rounded-sm align-super text-[10px] leading-none font-semibold text-primary underline-offset-2 hover:bg-primary/10 hover:underline focus-visible:outline-2 focus-visible:outline-ring"
								>
									{match.reference.replace(/^S/, "")}
								</button>
							) : (
								<Fragment key={`${offset}-${part}`}>{part}</Fragment>
							);
						})}
					</div>
					{!insufficient && answer.sources.length > 0 && (
						<div className="mt-7 border-t pt-5">
							<p className="mb-3 text-[10px] font-semibold uppercase tracking-[0.16em] text-muted-foreground">
								Sources · {answer.sources.length}
							</p>
							<div className="flex flex-wrap gap-2">
								{answer.sources.map((item) => (
									<button
										type="button"
										key={item.reference}
										onClick={() => setSource(item)}
										className="group flex min-w-0 basis-full items-center gap-3 rounded-lg border bg-muted/25 p-3 text-left transition-colors hover:bg-muted/70 focus-visible:outline-2 focus-visible:outline-ring sm:basis-[calc(50%-0.25rem)] sm:grow sm:max-w-[360px]"
									>
										<span className="flex size-8 shrink-0 items-center justify-center rounded bg-background text-[11px] font-semibold text-primary">
											{item.reference}
										</span>
										<div className="min-w-0 flex-1">
											<p className="truncate text-sm font-medium">
												{item.title || "Untitled document"}
											</p>
											<p className="mt-0.5 truncate text-xs text-muted-foreground">
												{item.section || `Section ${item.chunk_index + 1}`}
											</p>
										</div>
										<ArrowUpRight className="size-4 shrink-0 text-muted-foreground group-hover:text-primary" />
									</button>
								))}
							</div>
						</div>
					)}
					{insufficient && (
						<p className="mt-4 text-sm leading-6 text-muted-foreground">
							Try a more specific question, or add a document that covers this
							topic. Recently uploaded documents may still be processing.
						</p>
					)}
					<div className="mt-5 flex flex-wrap items-center justify-between gap-2 border-t pt-4">
						<p className="text-[11px] text-muted-foreground">
							Check the source material before relying on an answer.
						</p>
						<Button variant="ghost" size="sm" onClick={copy}>
							{copied ? <Check /> : <Copy />}
							{copied ? "Copied" : "Copy answer"}
						</Button>
					</div>
					{copyError && (
						<p aria-live="polite" className="text-xs text-muted-foreground">
							Copy isn’t available here. Select the answer text to copy it.
						</p>
					)}
				</CardContent>
			</Card>
			<Dialog
				open={source !== null}
				onOpenChange={(open) => {
					if (!open) setSource(null);
				}}
			>
				<DialogContent className="max-h-[85dvh] overflow-y-auto sm:max-w-2xl">
					<DialogHeader>
						<span className="mb-2 flex items-center gap-2 text-xs font-medium text-primary">
							<FileText className="size-4" />
							Source {source?.reference}
						</span>
						<DialogTitle className="pr-5 leading-7">
							{source?.title || "Untitled document"}
						</DialogTitle>
						<DialogDescription>
							{source?.section || "Document section"} · Original source text
						</DialogDescription>
					</DialogHeader>
					<div className="my-2 whitespace-pre-wrap rounded-xl border bg-muted/35 p-5 text-sm leading-7">
						{source?.text}
					</div>
					<p className="text-xs text-muted-foreground">
						This is the stored document section used to support the answer.
					</p>
				</DialogContent>
			</Dialog>
		</section>
	);
}
