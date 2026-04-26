import { apiFetch } from "./client";

export function login(
	email: string,
	password: string,
): Promise<{ token: string }> {
	return apiFetch("/auth/login", {
		method: "POST",
		body: JSON.stringify({ email, password }),
	});
}
