import { login } from "@/api/auth";
import { ApiError } from "@/api/client";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { useAuth } from "@/contexts/AuthContext";
import { useForm } from "@tanstack/react-form";
import { Briefcase } from "lucide-react";
import { useState } from "react";
import { Navigate, useNavigate } from "react-router-dom";
import { z } from "zod";

const schema = z.object({
	email: z.string().min(1, "Email is required").email("Invalid email"),
	password: z.string().min(1, "Password is required"),
});

export function LoginPage() {
	const { token, login: saveToken } = useAuth();
	const navigate = useNavigate();
	const [serverError, setServerError] = useState<string | null>(null);

	const form = useForm({
		defaultValues: { email: "", password: "" },
		onSubmit: async ({ value }) => {
			setServerError(null);
			try {
				const { token: t } = await login(value.email, value.password);
				saveToken(t);
				navigate("/dashboard");
			} catch (e) {
				setServerError(
					e instanceof ApiError ? e.message : "Login failed. Try again.",
				);
			}
		},
	});

	if (token) return <Navigate to="/dashboard" replace />;

	return (
		<div className="flex min-h-svh items-center justify-center bg-background px-4">
			<div className="w-full max-w-sm space-y-8">
				<div className="flex flex-col items-center gap-2 text-center">
					<div className="flex items-center gap-2">
						<Briefcase className="size-6" />
						<span className="text-xl font-semibold">AOP</span>
					</div>
					<p className="text-sm text-muted-foreground">
						Ansible Orchestration Platform
					</p>
				</div>

				<form
					onSubmit={(e) => {
						e.preventDefault();
						form.handleSubmit();
					}}
					className="space-y-4"
				>
					<form.Field
						name="email"
						validators={{
							onBlur: ({ value }) => {
								const r = schema.shape.email.safeParse(value);
								if (!r.success) return r.error.issues[0]?.message;
							},
						}}
					>
						{(field) => (
							<div className="space-y-1">
								<Label htmlFor="email">Email</Label>
								<Input
									id="email"
									type="email"
									autoComplete="email"
									value={field.state.value}
									onChange={(e) => field.handleChange(e.target.value)}
									onBlur={field.handleBlur}
									aria-invalid={field.state.meta.errors.length > 0}
								/>
								{field.state.meta.errors.length > 0 && (
									<p className="text-xs text-destructive">
										{String(field.state.meta.errors[0])}
									</p>
								)}
							</div>
						)}
					</form.Field>

					<form.Field
						name="password"
						validators={{
							onBlur: ({ value }) => {
								const r = schema.shape.password.safeParse(value);
								if (!r.success) return r.error.issues[0]?.message;
							},
						}}
					>
						{(field) => (
							<div className="space-y-1">
								<Label htmlFor="password">Password</Label>
								<Input
									id="password"
									type="password"
									autoComplete="current-password"
									value={field.state.value}
									onChange={(e) => field.handleChange(e.target.value)}
									onBlur={field.handleBlur}
									aria-invalid={field.state.meta.errors.length > 0}
								/>
								{field.state.meta.errors.length > 0 && (
									<p className="text-xs text-destructive">
										{String(field.state.meta.errors[0])}
									</p>
								)}
							</div>
						)}
					</form.Field>

					{serverError && (
						<p className="text-xs text-destructive">{serverError}</p>
					)}

					<form.Subscribe selector={(s) => s.isSubmitting}>
						{(isSubmitting) => (
							<Button type="submit" className="w-full" disabled={isSubmitting}>
								{isSubmitting ? "Signing in…" : "Sign in"}
							</Button>
						)}
					</form.Subscribe>
				</form>
			</div>
		</div>
	);
}
