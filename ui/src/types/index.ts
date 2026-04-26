export type JobStatus =
	| "pending"
	| "dispatched"
	| "running"
	| "success"
	| "failed"
	| "cancelled";

export function isTerminal(status: JobStatus) {
	return status === "success" || status === "failed" || status === "cancelled";
}

export interface Job {
	id: string;
	template_id: string;
	agent_id: string | null;
	workflow_run_id: string | null;
	status: JobStatus;
	extra_vars: Record<string, unknown>;
	facts: Record<string, unknown>;
	failure_reason: string | null;
	created_at: string;
	updated_at: string;
	started_at: string | null;
	ended_at: string | null;
}

export interface JobTemplate {
	id: string;
	name: string;
	description: string;
	project_id: string;
	playbook: string;
	credential_id: string | null;
	default_extra_vars: Record<string, unknown>;
	created_at: string;
	updated_at: string;
}

export interface Project {
	id: string;
	name: string;
	repo_url: string;
	branch: string;
	inventory_path: string;
	credential_id: string | null;
	sync_status: "pending" | "syncing" | "success" | "failed";
	last_synced_at: string | null;
	sync_error: string | null;
	created_at: string;
	updated_at: string;
}

export interface Agent {
	id: string;
	name: string;
	address: string;
	status: "online" | "offline";
	labels: Record<string, string>;
	capacity: number;
	running_jobs: number;
	last_heartbeat_at: string | null;
	registered_at: string;
	updated_at: string;
}

export interface JobLogLine {
	seq: number;
	line: string;
	stream: "stdout" | "stderr";
}
