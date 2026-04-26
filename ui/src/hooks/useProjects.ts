import { createProject, deleteProject, listProjects, syncProject } from "@/api/projects";
import { useAuth } from "@/contexts/AuthContext";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

export function useProjects() {
	const { token } = useAuth();
	return useQuery({
		queryKey: ["projects"],
		queryFn: () => listProjects(token!),
		enabled: !!token,
	});
}

export function useCreateProject() {
	const { token } = useAuth();
	const qc = useQueryClient();
	return useMutation({
		mutationFn: (data: Parameters<typeof createProject>[1]) =>
			createProject(token!, data),
		onSuccess: () => qc.invalidateQueries({ queryKey: ["projects"] }),
	});
}

export function useDeleteProject() {
	const { token } = useAuth();
	const qc = useQueryClient();
	return useMutation({
		mutationFn: (id: string) => deleteProject(token!, id),
		onSuccess: () => qc.invalidateQueries({ queryKey: ["projects"] }),
	});
}

export function useSyncProject() {
	const { token } = useAuth();
	const qc = useQueryClient();
	return useMutation({
		mutationFn: (id: string) => syncProject(token!, id),
		onSuccess: () => qc.invalidateQueries({ queryKey: ["projects"] }),
	});
}
