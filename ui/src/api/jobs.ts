import type { Job, JobLogLine, JobStatus } from "@/types";
import { apiFetch } from "./client";

export function listJobs(
	token: string,
	filters: { status?: JobStatus; template_id?: string } = {},
): Promise<Job[]> {
	const params = new URLSearchParams();
	if (filters.status) params.set("status", filters.status);
	if (filters.template_id) params.set("template_id", filters.template_id);
	const qs = params.toString();
	return apiFetch(`/jobs${qs ? `?${qs}` : ""}`, { token });
}

export function getJob(token: string, id: string): Promise<Job> {
	return apiFetch(`/jobs/${id}`, { token });
}

export function createJob(
	token: string,
	templateId: string,
	extraVars?: Record<string, unknown>,
): Promise<Job> {
	return apiFetch("/jobs", {
		method: "POST",
		token,
		body: JSON.stringify({ template_id: templateId, extra_vars: extraVars }),
	});
}

export function cancelJob(token: string, id: string): Promise<void> {
	return apiFetch(`/jobs/${id}/cancel`, { method: "POST", token });
}

export function getJobLogs(token: string, id: string): Promise<JobLogLine[]> {
	return apiFetch(`/jobs/${id}/logs`, { token });
}
