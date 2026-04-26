import type { Agent } from "@/types";
import { apiFetch } from "./client";

export function listAgents(token: string): Promise<Agent[]> {
	return apiFetch("/agents", { token });
}
