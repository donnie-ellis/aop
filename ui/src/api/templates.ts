import type { JobTemplate } from "@/types";
import { apiFetch } from "./client";

export function listTemplates(token: string): Promise<JobTemplate[]> {
	return apiFetch("/job-templates", { token });
}

export function getTemplate(token: string, id: string): Promise<JobTemplate> {
	return apiFetch(`/job-templates/${id}`, { token });
}

export function createTemplate(
	token: string,
	data: {
		name: string;
		description?: string;
		project_id: string;
		playbook: string;
		credential_id?: string;
		default_extra_vars?: Record<string, unknown>;
	},
): Promise<JobTemplate> {
	return apiFetch("/job-templates", {
		method: "POST",
		token,
		body: JSON.stringify(data),
	});
}

export function deleteTemplate(token: string, id: string): Promise<void> {
	return apiFetch(`/job-templates/${id}`, { method: "DELETE", token });
}
