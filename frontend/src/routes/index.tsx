import { createFileRoute } from "@tanstack/react-router";
import { WorkspaceApp } from "#/components/workspace";

export const Route = createFileRoute("/")({ component: Home });

function Home() {
	return <WorkspaceApp />;
}
