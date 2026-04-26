import type { Project } from "@/types";
import { apiFetch } from "./client";

export function listProjects(token: string): Promise<Project[]> {
	return apiFetch("/projects", { token });
}

export function createProject(
	token: string,
	data: {
		name: string;
		repo_url: string;
		branch?: string;
		inventory_path: string;
		credential_id?: string;
	},
): Promise<Project> {
	return apiFetch("/projects", {
		method: "POST",
		token,
		body: JSON.stringify(data),
	});
}

export function deleteProject(token: string, id: string): Promise<void> {
	return apiFetch(`/projects/${id}`, { method: "DELETE", token });
}

export function syncProject(token: string, id: string): Promise<void> {
	return apiFetch(`/projects/${id}/sync`, { method: "POST", token });
}
