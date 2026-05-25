import { LoginForm } from "#/components/login-form";
import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/")({ component: Home });

function Home() {
  return (
    <div className="flex min-h-screen w-full items-center justify-center bg-background">
      <div className="w-full max-w-md">
        <LoginForm />
      </div>
    </div>
  );
}
