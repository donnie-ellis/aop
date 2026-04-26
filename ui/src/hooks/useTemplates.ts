import { createTemplate, deleteTemplate, listTemplates } from "@/api/templates";
import { useAuth } from "@/contexts/AuthContext";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

export function useTemplates() {
	const { token } = useAuth();
	return useQuery({
		queryKey: ["templates"],
		queryFn: () => listTemplates(token!),
		enabled: !!token,
	});
}

export function useCreateTemplate() {
	const { token } = useAuth();
	const qc = useQueryClient();
	return useMutation({
		mutationFn: (data: Parameters<typeof createTemplate>[1]) =>
			createTemplate(token!, data),
		onSuccess: () => qc.invalidateQueries({ queryKey: ["templates"] }),
	});
}

export function useDeleteTemplate() {
	const { token } = useAuth();
	const qc = useQueryClient();
	return useMutation({
		mutationFn: (id: string) => deleteTemplate(token!, id),
		onSuccess: () => qc.invalidateQueries({ queryKey: ["templates"] }),
	});
}
