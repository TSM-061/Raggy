import { Check, FileText, LoaderCircle, UploadCloud } from "lucide-react";
import { useRef, useState } from "react";
import { Button } from "#/components/ui/button";
import {
	Dialog,
	DialogContent,
	DialogDescription,
	DialogHeader,
	DialogTitle,
} from "#/components/ui/dialog";
import {
	formatBytes,
	putUpload,
	SessionExpired,
	type StagedUpload,
	stageUpload,
} from "#/lib/api";
import { DEMO_MODE } from "#/lib/demo-api";

export function UploadDialog({
	open,
	onOpenChange,
	onUploaded,
	onExpired,
}: {
	open: boolean;
	onOpenChange: (open: boolean) => void;
	onUploaded: () => void;
	onExpired: () => void;
}) {
	const [file, setFile] = useState<File | null>(null);
	const [staged, setStaged] = useState<StagedUpload | null>(null);
	const [phase, setPhase] = useState<
		"idle" | "preparing" | "uploading" | "done"
	>("idle");
	const [error, setError] = useState("");
	const [dragging, setDragging] = useState(false);
	const input = useRef<HTMLInputElement>(null);
	const pending = phase === "preparing" || phase === "uploading";

	function selectFile(next?: File) {
		if (!next || pending) return;
		setError("");
		setStaged(null);
		setPhase("idle");
		setFile(null);
		if (!/\.(md|markdown)$/i.test(next.name)) {
			setError("Choose a Markdown document (.md or .markdown).");
			return;
		}
		if (!next.size) {
			setError("This file is empty. Choose a document with some content.");
			return;
		}
		if (next.size > 100 * 1024 * 1024) {
			setError("Choose a document smaller than 100 MB.");
			return;
		}
		setFile(next);
	}

	async function upload() {
		if (!file || pending) return;
		setError("");
		try {
			setPhase("preparing");
			const next = staged ?? (await stageUpload(file));
			setStaged(next);
			setPhase("uploading");
			await putUpload(next, file);
			setPhase("done");
			onUploaded();
		} catch (err) {
			setPhase("idle");
			if (err instanceof SessionExpired) {
				onExpired();
				return;
			}
			setError(
				err instanceof Error
					? err.message
					: "Could not upload your document. Please try again.",
			);
			// A pending record can exist even if transfer failed; refresh the list truthfully.
			onUploaded();
		}
	}

	function changeOpen(next: boolean) {
		if (pending) return;
		onOpenChange(next);
		if (!next) {
			setFile(null);
			setStaged(null);
			setPhase("idle");
			setError("");
		}
	}

	return (
		<Dialog open={open} onOpenChange={changeOpen}>
			<DialogContent showCloseButton={!pending} className="sm:max-w-lg">
				<DialogHeader>
					<DialogTitle>
						{phase === "done"
							? "Your document is on its way"
							: "Add a document to your workspace"}
					</DialogTitle>
					<DialogDescription>
						{phase === "done"
							? DEMO_MODE
								? "This is a simulated upload. Your file has not been sent anywhere."
								: "Your file has been uploaded. It will be processed in the background."
							: "Upload a document in Markdown format to start exploring its contents."}
					</DialogDescription>
				</DialogHeader>
				{phase === "done" ? (
					<div className="space-y-5 pt-3">
						<div className="flex items-center gap-3 rounded-xl border bg-secondary/40 p-4">
							<span className="flex size-10 items-center justify-center rounded-full bg-primary/10 text-primary">
								<Check className="size-5" />
							</span>
							<div className="min-w-0">
								<p className="truncate text-sm font-medium">{file?.name}</p>
								<p className="text-xs text-muted-foreground">
									{DEMO_MODE
										? "Simulated upload complete"
										: "Uploaded successfully"}
								</p>
							</div>
						</div>
						<p className="text-sm leading-6 text-muted-foreground">
							{DEMO_MODE
								? "The document appears in your preview list, but its contents are not indexed. Answers use the included sample document. Reload to reset your preview."
								: "New documents may take a little time to appear in answers. You can keep using your workspace while processing continues."}
						</p>
						<Button className="w-full" onClick={() => changeOpen(false)}>
							Back to workspace
						</Button>
					</div>
				) : (
					<div className="space-y-5 pt-3">
						<input
							ref={input}
							type="file"
							accept=".md,.markdown,text/markdown"
							aria-label="Choose Markdown document"
							className="sr-only"
							disabled={pending}
							onChange={(event) => {
								selectFile(event.target.files?.[0]);
								event.target.value = "";
							}}
						/>
						<button
							type="button"
							disabled={pending}
							onClick={() => input.current?.click()}
							onDragOver={(event) => {
								event.preventDefault();
								if (!pending) setDragging(true);
							}}
							onDragLeave={() => setDragging(false)}
							onDrop={(event) => {
								event.preventDefault();
								setDragging(false);
								selectFile(event.dataTransfer.files[0]);
							}}
							className={`flex w-full flex-col items-center rounded-xl border border-dashed px-5 py-9 text-center transition-colors outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:opacity-60 ${dragging ? "border-primary bg-primary/5" : "bg-muted/35 hover:bg-muted/70"}`}
						>
							<span className="mb-4 flex size-12 items-center justify-center rounded-xl border bg-background text-primary">
								<UploadCloud className="size-6" />
							</span>
							<span className="text-sm font-medium">
								Drop your document here, or browse files
							</span>
							<span className="mt-2 text-xs text-muted-foreground">
								Markdown files · up to 100 MB
							</span>
						</button>
						{file && (
							<div className="flex items-center gap-3 rounded-lg border p-3">
								<FileText className="size-5 shrink-0 text-primary" />
								<div className="min-w-0">
									<p className="truncate text-sm font-medium">{file.name}</p>
									<p className="text-xs text-muted-foreground">
										{formatBytes(file.size)}
									</p>
								</div>
								<Button
									variant="ghost"
									size="sm"
									className="ml-auto"
									disabled={pending}
									onClick={() => input.current?.click()}
								>
									Change
								</Button>
							</div>
						)}
						{error && (
							<p
								role="alert"
								className="rounded-lg bg-destructive/5 p-3 text-sm text-destructive"
							>
								{error}
							</p>
						)}
						<p className="text-xs leading-5 text-muted-foreground">
							Use a document with a title and subject in its frontmatter,
							followed by sections with Markdown headings.
						</p>
						<Button
							className="w-full"
							disabled={!file || pending}
							onClick={upload}
						>
							{pending ? (
								<LoaderCircle className="animate-spin" />
							) : (
								<UploadCloud />
							)}
							{phase === "preparing"
								? "Preparing upload…"
								: phase === "uploading"
									? "Uploading document…"
									: error && staged
										? "Retry upload"
										: "Upload document"}
						</Button>
						{pending && (
							<p
								aria-live="polite"
								className="text-center text-xs text-muted-foreground"
							>
								Keep this window open until your upload finishes.
							</p>
						)}
					</div>
				)}
			</DialogContent>
		</Dialog>
	);
}
