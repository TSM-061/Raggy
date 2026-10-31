import type { Upload } from "#/lib/api";

/** Displays completed chunks when the expected document total is known. */
export function ChunkProgress({ upload }: { upload: Upload }) {
	const total = upload.chunkTotal;
	if (total == null)
		return (
			<span className="text-[10px] text-muted-foreground">
				{upload.status === "pending"
					? "Awaiting upload"
					: "Waiting for chunk count"}
			</span>
		);
	const count = Math.min(total, Math.max(0, upload.chunksProcessed ?? 0));
	const percent = total === 0 ? 100 : Math.round((count / total) * 100);
	return (
		<div className="min-w-[130px] max-w-[220px] space-y-1.5">
			<div className="flex justify-between gap-3 text-[10px] text-muted-foreground">
				<span>
					{count} / {total} chunks
				</span>
				<span>{percent}%</span>
			</div>
			<div
				role="progressbar"
				aria-label={`Chunk processing for ${upload.originalName}`}
				aria-valuemin={0}
				aria-valuemax={100}
				aria-valuenow={percent}
				aria-valuetext={`${count} of ${total} chunks processed`}
				className="h-1.5 overflow-hidden rounded-full bg-primary/10"
			>
				<div
					className="h-full rounded-full bg-primary transition-[width] duration-300"
					style={{ width: `${percent}%` }}
				/>
			</div>
		</div>
	);
}
