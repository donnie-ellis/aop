import { cancelJob, createJob, getJob, getJobLogs, listJobs } from "@/api/jobs";
import { useAuth } from "@/contexts/AuthContext";
import type { JobStatus } from "@/types";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

export function useJobs(filters: { status?: JobStatus; template_id?: string } = {}) {
	const { token } = useAuth();
	return useQuery({
		queryKey: ["jobs", filters],
		queryFn: () => listJobs(token!, filters),
		enabled: !!token,
		refetchInterval: 5000,
	});
}

export function useJob(id: string) {
	const { token } = useAuth();
	return useQuery({
		queryKey: ["jobs", id],
		queryFn: () => getJob(token!, id),
		enabled: !!token && !!id,
		refetchInterval: (query) => {
			const status = query.state.data?.status;
			if (!status) return 3000;
			return status === "running" || status === "dispatched" || status === "pending"
				? 3000
				: false;
		},
	});
}

export function useJobLogs(id: string, enabled: boolean) {
	const { token } = useAuth();
	return useQuery({
		queryKey: ["jobs", id, "logs"],
		queryFn: () => getJobLogs(token!, id),
		enabled: !!token && !!id && enabled,
		refetchInterval: (query) => {
			// keep polling logs while job is running
			return query.state.data !== undefined ? false : 3000;
		},
	});
}

export function useCreateJob() {
	const { token } = useAuth();
	const qc = useQueryClient();
	return useMutation({
		mutationFn: ({
			templateId,
			extraVars,
		}: { templateId: string; extraVars?: Record<string, unknown> }) =>
			createJob(token!, templateId, extraVars),
		onSuccess: () => qc.invalidateQueries({ queryKey: ["jobs"] }),
	});
}

export function useCancelJob() {
	const { token } = useAuth();
	const qc = useQueryClient();
	return useMutation({
		mutationFn: (id: string) => cancelJob(token!, id),
		onSuccess: () => qc.invalidateQueries({ queryKey: ["jobs"] }),
	});
}
