import {
	keepPreviousData,
	useQuery,
	useQueryClient,
} from "@tanstack/react-query";
import {
	ArrowDown,
	ArrowRight,
	ArrowUp,
	BookOpen,
	ChevronLeft,
	ChevronRight,
	CircleHelp,
	FileText,
	Layers2,
	LoaderCircle,
	LogOut,
	MessageSquare,
	Plus,
	RefreshCw,
	Search,
	ShieldCheck,
	UploadCloud,
} from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { AnswerPanel } from "#/components/answer-panel";
import { ChunkProgress } from "#/components/chunk-progress";
import { LoginForm } from "#/components/login-form";
import { Badge } from "#/components/ui/badge";
import { Button } from "#/components/ui/button";
import { Card, CardContent } from "#/components/ui/card";
import { Textarea } from "#/components/ui/textarea";
import { UploadDialog } from "#/components/upload-dialog";
import {
	type Answer,
	api,
	askQuestion,
	formatBytes,
	listUploads,
	SessionExpired,
	type Upload,
	type UploadPage,
} from "#/lib/api";
import { DEMO_MODE, SAMPLE_QUESTION } from "#/lib/demo-api";
import { useUploadEvents } from "#/lib/use-upload-events";
import { cn } from "#/lib/utils";

function Brand({ inverse = false }: { inverse?: boolean }) {
	return (
		<div className={cn("flex items-center gap-2.5", inverse && "text-white")}>
			<span
				className={cn(
					"flex size-9 items-center justify-center rounded-xl bg-primary text-primary-foreground",
					inverse && "bg-white/15",
				)}
			>
				<Layers2 className="size-5" />
			</span>
			<span className="text-2xl font-bold tracking-[-0.07em]">
				raggy<span className="text-[#c98346]">.</span>
			</span>
		</div>
	);
}

export function WorkspaceApp() {
	const [mounted, setMounted] = useState(false);
	const [signedOut, setSignedOut] = useState(false);
	const [page, setPage] = useState(1);
	const client = useQueryClient();
	useEffect(() => setMounted(true), []);
	const [liveConnected, setLiveConnected] = useState(false);
	const uploads = useQuery({
		queryKey: ["uploads", page],
		queryFn: ({ signal }) => listUploads(page, signal),
		enabled: mounted && !signedOut,
		placeholderData: keepPreviousData,
		retry: false,
		refetchInterval: (query) =>
			!(query.state.error instanceof SessionExpired)
				? liveConnected
					? 30000
					: 5000
				: false,
	});

	const connected = useUploadEvents(
		mounted && !signedOut && !(uploads.error instanceof SessionExpired),
	);
	useEffect(() => setLiveConnected(connected), [connected]);

	function leaveSession() {
		setSignedOut(true);
		setPage(1);
		void client.cancelQueries({ queryKey: ["uploads"] });
		client.removeQueries({ queryKey: ["uploads"] });
	}
	function signedIn() {
		setPage(1);
		setSignedOut(false);
		void client.resetQueries({ queryKey: ["uploads"] });
	}

	if (signedOut || uploads.error instanceof SessionExpired)
		return <LoginScreen onSuccess={signedIn} />;
	if (!mounted || (!uploads.data && uploads.isPending))
		return (
			<div className="flex min-h-dvh flex-col items-center justify-center gap-6 bg-background">
				<Brand />
				<p
					aria-live="polite"
					className="flex items-center gap-2 text-sm text-muted-foreground"
				>
					<LoaderCircle className="size-4 animate-spin" />
					Opening your workspace…
				</p>
			</div>
		);
	if (!uploads.data)
		return (
			<div className="flex min-h-dvh items-center justify-center p-6">
				<Card className="w-full max-w-md">
					<CardContent className="space-y-5 p-8">
						<Brand />
						<h1 className="text-xl font-semibold">
							We couldn’t reach your workspace
						</h1>
						<p role="alert" className="text-sm leading-6 text-muted-foreground">
							Check your connection and make sure the Raggy services are
							running, then try again.
						</p>
						<Button onClick={() => void uploads.refetch()}>
							<RefreshCw />
							Try again
						</Button>
					</CardContent>
				</Card>
			</div>
		);

	return (
		<Workspace
			data={uploads.data}
			page={page}
			onPage={setPage}
			refreshing={uploads.isFetching}
			pagePending={uploads.isPlaceholderData}
			error={uploads.error?.message}
			onRefresh={() => void client.invalidateQueries({ queryKey: ["uploads"] })}
			onExpired={leaveSession}
		/>
	);
}

function LoginScreen({ onSuccess }: { onSuccess: () => void }) {
	return (
		<main className="grid min-h-dvh lg:grid-cols-[1.05fr_1fr]">
			<div className="login-story relative hidden flex-col overflow-hidden bg-[#183d35] p-12 text-white lg:flex xl:p-16">
				<Brand inverse />
				<div className="relative z-10 my-auto max-w-lg py-20">
					<span className="mb-7 inline-flex items-center gap-2 rounded-full border border-white/20 px-3 py-1.5 text-[11px] tracking-wide text-white/75">
						<span className="size-1.5 rounded-full bg-[#d7aa7b]" />A clearer
						view of your knowledge
					</span>
					<h1 className="text-5xl font-medium leading-[1.16] tracking-[-0.045em] xl:text-6xl">
						Your documents.
						<br />
						Their answers.
						<br />
						<span className="text-[#b8cdbf]">Connected.</span>
					</h1>
					<p className="mt-7 max-w-sm text-sm leading-7 text-white/60">
						Turn documents into a workspace you can ask questions. Every answer
						brings you back to the source.
					</p>
					<div className="mt-10 flex items-center gap-3 border-t border-white/15 pt-7 text-xs text-white/65">
						<BookOpen className="size-4 text-[#d7aa7b]" />
						Built on your documents. Backed by references.
					</div>
				</div>
				<p className="text-[11px] text-white/40">
					A little less searching. A little more understanding.
				</p>
			</div>
			<div className="flex flex-col px-6 py-8 sm:px-12">
				<div className="lg:hidden">
					<Brand />
				</div>
				<div className="mx-auto my-auto w-full max-w-[360px] py-16">
					<p className="eyebrow mb-3">Your knowledge workspace</p>
					<h2 className="mb-3 text-3xl font-semibold tracking-tight">
						{DEMO_MODE ? "Explore the preview." : "Welcome back."}
					</h2>
					<p className="mb-9 text-sm leading-6 text-muted-foreground">
						{DEMO_MODE
							? "Sample documents and example answers. No account or AI connection needed."
							: "Sign in to explore your documents and find answers with context."}
					</p>
					{DEMO_MODE ? (
						<Button className="w-full" onClick={onSuccess}>
							Open sample workspace
							<ArrowRight />
						</Button>
					) : (
						<LoginForm onSuccess={onSuccess} />
					)}
					<p className="mt-7 text-center text-xs leading-5 text-muted-foreground">
						{DEMO_MODE
							? "Preview changes reset when you reload the page."
							: "Use the account provided by your workspace administrator."}
					</p>
				</div>
				<div className="flex items-center justify-center gap-2 text-[11px] text-muted-foreground">
					<ShieldCheck className="size-3.5" />
					Your session stays in this browser.
				</div>
			</div>
		</main>
	);
}

type WorkspaceProps = {
	data: UploadPage;
	page: number;
	onPage: (page: number) => void;
	refreshing: boolean;
	pagePending: boolean;
	error?: string;
	onRefresh: () => void;
	onExpired: () => void;
};

function Workspace({
	data,
	page,
	onPage,
	refreshing,
	pagePending,
	error,
	onRefresh,
	onExpired,
}: WorkspaceProps) {
	const [view, setView] = useState<"ask" | "documents">("ask");
	const [uploadOpen, setUploadOpen] = useState(false);
	const [signingOut, setSigningOut] = useState(false);
	const [sessionError, setSessionError] = useState("");
	async function signOut() {
		setSigningOut(true);
		setSessionError("");
		try {
			await api("/api/auth/signout", { method: "POST" });
			onExpired();
		} catch {
			setSessionError("Could not sign out. Please try again.");
		} finally {
			setSigningOut(false);
		}
	}

	const nav = (
		<>
			<NavButton
				active={view === "ask"}
				onClick={() => setView("ask")}
				icon={<MessageSquare className="size-4" />}
			>
				Ask workspace
			</NavButton>
			<NavButton
				active={view === "documents"}
				onClick={() => setView("documents")}
				icon={<FileText className="size-4" />}
			>
				Documents
				<span className="ml-auto rounded bg-foreground/5 px-1.5 py-0.5 text-[10px] tabular-nums">
					{data.total}
				</span>
			</NavButton>
		</>
	);
	return (
		<div className="min-h-dvh lg:grid lg:grid-cols-[232px_minmax(0,1fr)]">
			<aside className="sticky top-0 hidden h-dvh flex-col border-r bg-sidebar p-5 lg:flex">
				<div className="px-2 pb-9 pt-2">
					<Brand />
				</div>
				<div className="mb-7 flex items-center gap-3 rounded-xl border bg-background p-3">
					<span className="flex size-8 items-center justify-center rounded-lg bg-[#ebe9df] text-xs font-semibold text-[#605c46]">
						W
					</span>
					<div>
						<p className="text-xs font-semibold">My workspace</p>
						<p className="mt-0.5 text-[10px] text-muted-foreground">
							Document knowledge
						</p>
					</div>
				</div>
				<p className="eyebrow mb-3 px-3 text-[9px]">Workspace</p>
				<nav aria-label="Main navigation" className="space-y-1">
					{nav}
				</nav>
				<div className="mt-auto space-y-5 pt-8">
					<div className="rounded-xl border border-primary/10 bg-primary/[0.035] p-4">
						<BookOpen className="mb-3 size-5 text-primary/75" />
						<p className="text-xs font-semibold">Knowledge starts here</p>
						<p className="mt-2 text-[11px] leading-5 text-muted-foreground">
							Add a document, ask a question, and follow the evidence.
						</p>
						<button
							type="button"
							onClick={() => setUploadOpen(true)}
							className="mt-3 flex items-center gap-1.5 text-xs font-medium text-primary hover:underline"
						>
							Upload a document
							<ArrowRight className="size-3" />
						</button>
					</div>
					<div className="flex items-center gap-2 border-t px-1 pt-4">
						<span className="flex size-8 items-center justify-center rounded-full border bg-background text-[11px] font-medium">
							ME
						</span>
						<span className="flex-1 text-xs font-medium">Your account</span>
						<Button
							variant="ghost"
							size="icon-sm"
							aria-label="Sign out"
							disabled={signingOut}
							onClick={signOut}
						>
							<LogOut className="size-4 text-muted-foreground" />
						</Button>
					</div>
				</div>
			</aside>
			<div className="min-w-0">
				<header className="flex h-[73px] items-center justify-between gap-3 border-b bg-background/90 px-5 sm:px-8 xl:px-12">
					<div className="lg:hidden">
						<Brand />
					</div>
					<div className="hidden items-center gap-2 text-xs text-muted-foreground lg:flex">
						Workspace
						<ChevronRight className="size-3" />
						<span className="font-medium text-foreground">
							{view === "ask" ? "Ask workspace" : "Documents"}
						</span>
					</div>
					<div className="flex items-center gap-3">
						<span className="hidden items-center gap-1.5 text-[11px] text-muted-foreground sm:flex">
							<span className="size-1.5 rounded-full bg-primary/60" />
							Workspace documents
						</span>
						<Button size="sm" onClick={() => setUploadOpen(true)}>
							<Plus />
							Upload document
						</Button>
						<Button
							className="lg:hidden"
							variant="ghost"
							size="icon-sm"
							aria-label="Sign out"
							onClick={signOut}
							disabled={signingOut}
						>
							<LogOut />
						</Button>
					</div>
				</header>
				<nav
					aria-label="Mobile navigation"
					className="flex gap-2 border-b bg-background p-3 lg:hidden"
				>
					{nav}
				</nav>
				<main className="mx-auto max-w-[1440px] px-5 py-8 sm:px-8 sm:py-10 xl:px-12">
					{DEMO_MODE && (
						<div
							role="note"
							className="mb-7 flex flex-wrap items-center gap-x-3 gap-y-1 rounded-xl border border-amber-200 bg-amber-50 px-4 py-3 text-xs leading-5 text-amber-950"
						>
							<Badge
								variant="outline"
								className="shrink-0 border-amber-300 bg-white/70 text-amber-900"
							>
								Sample-data preview
							</Badge>
							<span>
								Example answers, no AI calls. Uploads are simulated; files stay
								in your browser. Changes reset on reload.
							</span>
						</div>
					)}
					{(error || sessionError) && (
						<div
							role="alert"
							className="mb-6 flex flex-wrap items-center justify-between gap-2 rounded-lg border border-destructive/15 bg-destructive/5 px-4 py-3 text-sm text-destructive"
						>
							{sessionError || error}
							<Button
								variant="ghost"
								size="sm"
								onClick={sessionError ? signOut : onRefresh}
							>
								Try again
							</Button>
						</div>
					)}
					{view === "ask" ? (
						<AskWorkspace
							total={data.total}
							uploads={data.uploads}
							onUpload={() => setUploadOpen(true)}
							onDocuments={() => setView("documents")}
							onExpired={onExpired}
						/>
					) : (
						<Documents
							data={data}
							page={page}
							onPage={onPage}
							pending={pagePending || refreshing}
							onUpload={() => setUploadOpen(true)}
							onRefresh={onRefresh}
						/>
					)}
				</main>
			</div>
			<UploadDialog
				open={uploadOpen}
				onOpenChange={setUploadOpen}
				onExpired={onExpired}
				onUploaded={() => {
					onPage(1);
					onRefresh();
				}}
			/>
		</div>
	);
}

function NavButton({
	active,
	icon,
	children,
	onClick,
}: {
	active: boolean;
	icon: React.ReactNode;
	children: React.ReactNode;
	onClick: () => void;
}) {
	return (
		<button
			type="button"
			aria-current={active ? "page" : undefined}
			onClick={onClick}
			className={cn(
				"flex min-h-10 w-full items-center gap-3 rounded-lg px-3 py-2.5 text-left text-xs font-medium transition-colors focus-visible:outline-2 focus-visible:outline-ring",
				active
					? "bg-primary/8 text-primary"
					: "text-muted-foreground hover:bg-muted hover:text-foreground",
			)}
		>
			{icon}
			{children}
		</button>
	);
}

function AskWorkspace({
	total,
	uploads,
	onUpload,
	onDocuments,
	onExpired,
}: {
	total: number;
	uploads: Upload[];
	onUpload: () => void;
	onDocuments: () => void;
	onExpired: () => void;
}) {
	const [question, setQuestion] = useState("");
	const [submitted, setSubmitted] = useState("");
	const [answer, setAnswer] = useState<Answer | null>(null);
	const [pending, setPending] = useState(false);
	const [error, setError] = useState("");
	const request = useRef<AbortController | null>(null);
	const textarea = useRef<HTMLTextAreaElement>(null);
	useEffect(() => () => request.current?.abort(), []);

	async function ask(
		event: React.SubmitEvent<HTMLFormElement> | null,
		sampleQuestion?: string,
	) {
		event?.preventDefault();
		const nextQuestion = sampleQuestion ?? question;
		if (!nextQuestion.trim() || pending) return;
		request.current?.abort();
		const controller = new AbortController();
		request.current = controller;
		const value = nextQuestion.trim();
		setSubmitted(value);
		setPending(true);
		setAnswer(null);
		setError("");
		try {
			const result = await askQuestion(value, controller.signal);
			if (!controller.signal.aborted) setAnswer(result);
		} catch (err) {
			if (controller.signal.aborted) return;
			if (err instanceof SessionExpired) {
				onExpired();
				return;
			}
			setError(
				err instanceof Error
					? err.message
					: "Could not find an answer. Please try again.",
			);
		} finally {
			if (!controller.signal.aborted) setPending(false);
		}
	}
	function cancel() {
		request.current?.abort();
		setPending(false);
	}

	return (
		<>
			<div className="mb-9 flex flex-wrap items-end justify-between gap-5">
				<div>
					<p className="eyebrow mb-3">Ask · Discover · Understand</p>
					<h1 className="text-[30px] font-semibold leading-tight tracking-[-0.04em] sm:text-[38px]">
						A clearer view of your documents.
					</h1>
					<p className="mt-3 text-sm leading-6 text-muted-foreground">
						Ask a question. Find the context. Go straight to the source.
					</p>
				</div>
				<Badge
					variant="outline"
					className="gap-2 bg-background px-3 py-1.5 text-[11px] font-normal"
				>
					<Layers2 className="size-3.5" />
					{total} {total === 1 ? "document" : "documents"} in your workspace
				</Badge>
			</div>
			<div className="grid items-start gap-6 xl:grid-cols-[minmax(0,1fr)_290px]">
				<div className="min-w-0 space-y-7">
					<Card className="gap-0 overflow-hidden border-primary/20 py-0 shadow-[0_4px_20px_-10px_#183d3520]">
						<form onSubmit={ask}>
							<div className="flex items-center gap-2 px-5 pt-5 text-xs font-medium text-primary">
								<Search className="size-3.5" />
								<label htmlFor="question">What would you like to know?</label>
							</div>
							<Textarea
								ref={textarea}
								id="question"
								value={question}
								onChange={(event) => setQuestion(event.target.value)}
								placeholder="Ask about requirements, risks, or recommendations…"
								disabled={pending}
								maxLength={5000}
								className="min-h-[122px] resize-none border-0 bg-transparent px-5 py-4 text-base shadow-none focus-visible:ring-0 sm:text-base"
								onKeyDown={(event) => {
									if (
										event.key === "Enter" &&
										(event.metaKey || event.ctrlKey)
									) {
										event.preventDefault();
										event.currentTarget.form?.requestSubmit();
									}
								}}
							/>
							<div className="flex items-center justify-between gap-3 border-t bg-muted/25 px-4 py-3">
								<span className="flex items-center gap-1.5 text-[11px] text-muted-foreground">
									<Layers2 className="size-3.5" />
									All workspace documents
								</span>
								{pending ? (
									<Button
										type="button"
										size="sm"
										variant="outline"
										onClick={cancel}
									>
										Cancel
									</Button>
								) : (
									<Button
										type="submit"
										size="sm"
										disabled={!question.trim() || total === 0}
									>
										Ask question
										<ArrowUp className="size-3.5" />
									</Button>
								)}
							</div>
						</form>
					</Card>
					{DEMO_MODE && !answer && (
						<div className="flex flex-wrap items-center justify-between gap-4 rounded-xl border border-primary/15 bg-primary/5 p-4">
							<div className="min-w-0 flex-1">
								<p className="text-xs font-semibold text-primary">
									See an annotated example
								</p>
								<p className="mt-1 text-xs leading-6 text-muted-foreground">
									{SAMPLE_QUESTION}
								</p>
							</div>
							<Button
								variant="outline"
								size="sm"
								disabled={pending}
								onClick={() => {
									setQuestion(SAMPLE_QUESTION);
									void ask(null, SAMPLE_QUESTION);
								}}
							>
								Try sample question
								<ArrowRight />
							</Button>
						</div>
					)}
					{error && (
						<div
							role="alert"
							className="rounded-xl border border-destructive/20 bg-destructive/5 p-4 text-sm text-destructive"
						>
							{error}
							<p className="mt-2 text-xs">
								Your question is still above. Submit it again to retry.
							</p>
						</div>
					)}
					{pending && (
						<Card>
							<CardContent className="flex items-center gap-4 p-6">
								<LoaderCircle className="size-5 animate-spin text-primary" />
								<div aria-live="polite">
									<p className="text-sm font-medium">
										Finding the relevant context…
									</p>
									<p className="mt-1 text-xs text-muted-foreground">
										Reading your documents and gathering supporting sources.
									</p>
								</div>
							</CardContent>
						</Card>
					)}
					{answer && !pending && (
						<AnswerPanel
							key={submitted + answer.response}
							answer={answer}
							question={submitted}
						/>
					)}
					{!answer && !pending && !error && (
						<div className="space-y-7">
							<div>
								<p className="mb-3 text-[11px] text-muted-foreground">
									A few starting points
								</p>
								<div className="grid gap-2 sm:grid-cols-3">
									{[
										["Get the overview", "Summarise the key information."],
										["Look a little closer", "Which risks need attention?"],
										["Connect the dots", "Compare the proposed solutions."],
									].map(([title, value]) => (
										<button
											type="button"
											key={title}
											onClick={() => {
												setQuestion(value);
												textarea.current?.focus();
											}}
											className="group flex flex-col rounded-xl border bg-background p-4 text-left transition-colors hover:border-primary/30 hover:bg-primary/[0.02] focus-visible:outline-2 focus-visible:outline-ring"
										>
											<span className="mb-3 flex w-full items-center justify-between text-[10px] text-muted-foreground">
												{title}
												<ArrowUp className="size-3 rotate-45 opacity-40 group-hover:opacity-100" />
											</span>
											<span className="text-xs leading-5">{value}</span>
										</button>
									))}
								</div>
							</div>
							<div className="rounded-xl border border-dashed px-6 py-9 text-center sm:px-10">
								<span className="mx-auto mb-4 flex size-11 items-center justify-center rounded-xl border bg-background text-primary/70">
									<BookOpen className="size-5" />
								</span>
								<h2 className="text-sm font-semibold">
									{total
										? "Good questions start with good context."
										: "Your next answer starts with a document."}
								</h2>
								<p className="mx-auto mt-2 max-w-sm text-xs leading-6 text-muted-foreground">
									{total
										? "Explore your documents with a question above. Answers include references so you can see where the information came from."
										: "Upload a document, then ask about what matters to you. Your documents become the starting point for every answer."}
								</p>
								{!total && (
									<Button
										variant="outline"
										size="sm"
										className="mt-5"
										onClick={onUpload}
									>
										<Plus />
										Upload your first document
									</Button>
								)}
							</div>
						</div>
					)}
					<p className="flex items-start justify-center gap-1.5 text-center text-[10px] leading-5 text-muted-foreground">
						<ShieldCheck className="mt-0.5 size-3.5 shrink-0" />
						Answers use your workspace context. Always review the original
						sources.
					</p>
				</div>
				<aside className="space-y-5">
					<Card className="gap-0 py-0 shadow-none">
						<div className="flex items-center justify-between border-b p-4">
							<h2 className="text-xs font-semibold">Your documents</h2>
							<Badge variant="secondary" className="text-[10px] tabular-nums">
								{total}
							</Badge>
						</div>
						<div className="p-4">
							{uploads.length ? (
								<div className="space-y-4">
									{uploads.slice(0, 4).map((item) => (
										<div key={item.id} className="flex items-start gap-3">
											<span className="mt-0.5 flex size-8 shrink-0 items-center justify-center rounded-lg bg-muted">
												<FileText className="size-4 text-primary/70" />
											</span>
											<div className="min-w-0">
												<p
													className="truncate text-xs font-medium"
													title={item.originalName}
												>
													{item.originalName}
												</p>
												<div className="mt-1.5 flex items-center gap-2 text-[10px] text-muted-foreground">
													<span>{formatBytes(item.sizeBytes)}</span>
													<span>·</span>
													<span>{statusLabel(item.status)}</span>
												</div>
												<div className="mt-2">
													<ChunkProgress upload={item} />
												</div>
											</div>
										</div>
									))}
								</div>
							) : (
								<p className="py-3 text-xs leading-6 text-muted-foreground">
									No documents yet. Add your first document to give your
									questions some context.
								</p>
							)}
							<Button
								variant="outline"
								size="sm"
								className="mt-5 w-full text-xs"
								onClick={total ? onDocuments : onUpload}
							>
								{total ? "View all documents" : "Add a document"}
								{total ? <ArrowRight /> : <Plus />}
							</Button>
						</div>
					</Card>
					<div className="rounded-xl bg-[#eeeee6] p-5">
						<CircleHelp className="mb-3 size-5 text-[#747654]" />
						<h2 className="text-xs font-semibold text-[#4f543b]">
							Follow the evidence
						</h2>
						<p className="mt-2 text-xs leading-6 text-[#73765e]">
							Click a source reference in an answer to read the original
							document section. A little context goes a long way.
						</p>
					</div>
					<p className="px-1 text-[10px] leading-5 text-muted-foreground">
						Progress updates as document chunks are embedded. Documents are
						ready once all expected chunks have completed.
					</p>
				</aside>
			</div>
		</>
	);
}

function statusLabel(status: string) {
	return status === "pending"
		? "Awaiting upload"
		: status === "uploaded"
			? "Uploaded"
			: status === "processing"
				? "Processing"
				: status === "processed"
					? "Ready"
					: status;
}

function Documents({
	data,
	page,
	onPage,
	pending,
	onUpload,
	onRefresh,
}: {
	data: UploadPage;
	page: number;
	onPage: (page: number) => void;
	pending: boolean;
	onUpload: () => void;
	onRefresh: () => void;
}) {
	const pages = Math.max(1, Math.ceil(data.total / data.pageSize));
	return (
		<>
			<div className="mb-8 flex flex-wrap items-end justify-between gap-4">
				<div>
					<p className="eyebrow mb-3">Your source material</p>
					<h1 className="text-3xl font-semibold tracking-tight">Documents</h1>
					<p className="mt-3 text-sm text-muted-foreground">
						A home for the documents behind your answers.
					</p>
				</div>
				<Button
					variant="outline"
					size="sm"
					onClick={onRefresh}
					disabled={pending}
				>
					<RefreshCw className={pending ? "animate-spin" : ""} />
					Refresh
				</Button>
			</div>
			<Card className="gap-0 overflow-hidden py-0 shadow-none">
				<div className="flex items-center justify-between border-b px-5 py-4">
					<h2 className="text-sm font-semibold">
						All documents{" "}
						<span className="ml-2 text-xs font-normal text-muted-foreground">
							{data.total}
						</span>
					</h2>
					<span className="text-[11px] text-muted-foreground">
						Markdown documents
					</span>
				</div>
				{data.uploads.length ? (
					<div className="overflow-x-auto">
						<table className="w-full min-w-[580px] text-left">
							<thead className="bg-muted/35 text-[10px] uppercase tracking-wider text-muted-foreground">
								<tr>
									<th className="px-5 py-3 font-medium" scope="col">
										Document name
									</th>
									<th className="px-4 py-3 font-medium" scope="col">
										Status
									</th>
									<th className="px-4 py-3 font-medium" scope="col">
										Progress
									</th>
									<th className="px-4 py-3 font-medium" scope="col">
										Size
									</th>
									<th className="px-5 py-3 font-medium" scope="col">
										Added
									</th>
								</tr>
							</thead>
							<tbody>
								{data.uploads.map((item) => (
									<tr key={item.id} className="border-t text-xs">
										<td className="max-w-[380px] px-5 py-5">
											<span className="flex items-center gap-3">
												<FileText className="size-4 shrink-0 text-primary/65" />
												<span className="truncate" title={item.originalName}>
													{item.originalName}
												</span>
											</span>
										</td>
										<td className="px-4 py-5">
											<Badge
												variant="outline"
												className={cn(
													"gap-1.5 whitespace-nowrap text-[10px] font-normal",
													item.status === "pending"
														? "border-amber-200 bg-amber-50 text-amber-800"
														: "border-primary/15 bg-primary/5 text-primary",
												)}
											>
												<span className="size-1 rounded-full bg-current" />
												{statusLabel(item.status)}
											</Badge>
										</td>
										<td className="px-4 py-5">
											<ChunkProgress upload={item} />
										</td>
										<td className="whitespace-nowrap px-4 py-5 text-muted-foreground">
											{formatBytes(item.sizeBytes)}
										</td>
										<td className="whitespace-nowrap px-5 py-5 text-muted-foreground">
											{new Intl.DateTimeFormat("en-AU", {
												day: "numeric",
												month: "short",
												year: "numeric",
											}).format(new Date(item.createdAt))}
										</td>
									</tr>
								))}
							</tbody>
						</table>
					</div>
				) : (
					<div className="px-6 py-20 text-center">
						<UploadCloud className="mx-auto mb-5 size-8 text-primary/50" />
						<h3 className="text-base font-semibold">
							Make room for your first document.
						</h3>
						<p className="mx-auto mt-2 max-w-sm text-sm leading-6 text-muted-foreground">
							Add a Markdown document and start building a workspace you can
							explore.
						</p>
						<Button className="mt-6" onClick={onUpload}>
							<Plus />
							Upload document
						</Button>
					</div>
				)}
				{data.total > 0 && (
					<div className="flex flex-wrap items-center justify-between gap-3 border-t px-5 py-4">
						<p className="text-xs text-muted-foreground">
							Page {page} of {pages}
						</p>
						<div className="flex gap-2">
							<Button
								variant="outline"
								size="sm"
								disabled={page === 1 || pending}
								onClick={() => onPage(page - 1)}
								aria-label="Previous page"
							>
								<ChevronLeft />
							</Button>
							<Button
								variant="outline"
								size="sm"
								disabled={page >= pages || pending}
								onClick={() => onPage(page + 1)}
								aria-label="Next page"
							>
								<ChevronRight />
							</Button>
						</div>
					</div>
				)}
			</Card>
			<p className="mt-5 flex items-start gap-2 text-xs leading-6 text-muted-foreground">
				<ArrowDown className="mt-1 size-3.5 shrink-0" />
				Chunk counts update automatically. Ready means all expected chunks have
				been persisted.
			</p>
		</>
	);
}
