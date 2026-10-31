import { ArrowRight, LoaderCircle } from "lucide-react";
import { useState } from "react";
import { Button } from "#/components/ui/button";
import { Input } from "#/components/ui/input";
import { Label } from "#/components/ui/label";
import { api } from "#/lib/api";

export function LoginForm({ onSuccess }: { onSuccess: () => void }) {
	const [pending, setPending] = useState(false);
	const [error, setError] = useState("");
	async function submit(event: React.SubmitEvent<HTMLFormElement>) {
		event.preventDefault();
		setError("");
		setPending(true);
		const data = Object.fromEntries(new FormData(event.currentTarget));
		try {
			await api("/api/auth/signin", {
				method: "POST",
				headers: { "Content-Type": "application/json" },
				body: JSON.stringify(data),
			});
			onSuccess();
		} catch (err) {
			setError(
				err instanceof Error
					? err.message
					: "Could not sign in. Please try again.",
			);
		} finally {
			setPending(false);
		}
	}
	return (
		<form onSubmit={submit} className="space-y-5">
			<div className="space-y-2">
				<Label htmlFor="username">Username</Label>
				<Input
					autoComplete="username"
					id="username"
					name="username"
					placeholder="Your username"
					required
					className="h-11"
				/>
			</div>
			<div className="space-y-2">
				<Label htmlFor="password">Password</Label>
				<Input
					autoComplete="current-password"
					id="password"
					type="password"
					name="password"
					placeholder="Your password"
					required
					className="h-11"
				/>
			</div>
			{error && (
				<p
					role="alert"
					className="rounded-lg bg-destructive/5 p-3 text-sm text-destructive"
				>
					{error}
				</p>
			)}
			<Button type="submit" disabled={pending} className="h-11 w-full">
				{pending && <LoaderCircle className="animate-spin" />}
				{pending ? "Signing in…" : "Sign in to your workspace"}
				{!pending && <ArrowRight />}
			</Button>
		</form>
	);
}
