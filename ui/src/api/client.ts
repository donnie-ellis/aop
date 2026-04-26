const API_BASE = import.meta.env.VITE_API_URL ?? "http://localhost:8080";

export class ApiError extends Error {
	readonly status: number;
	constructor(status: number, message: string) {
		super(message);
		this.status = status;
		this.name = "ApiError";
	}
}

type Opts = RequestInit & { token?: string | null };

export async function apiFetch<T>(path: string, opts: Opts = {}): Promise<T> {
	const { token, ...init } = opts;

	const headers: Record<string, string> = {
		"Content-Type": "application/json",
		...(token ? { Authorization: `Bearer ${token}` } : {}),
		...(init.headers as Record<string, string>),
	};

	const res = await fetch(`${API_BASE}${path}`, { ...init, headers });

	if (!res.ok) {
		let message = res.statusText;
		try {
			const body = await res.json();
			message = body.error ?? message;
		} catch {}
		throw new ApiError(res.status, message);
	}

	if (res.status === 204) return undefined as T;
	return res.json();
}
