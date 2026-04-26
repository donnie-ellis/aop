import { listAgents } from "@/api/agents";
import { useAuth } from "@/contexts/AuthContext";
import { useQuery } from "@tanstack/react-query";

export function useAgents() {
	const { token } = useAuth();
	return useQuery({
		queryKey: ["agents"],
		queryFn: () => listAgents(token!),
		enabled: !!token,
		refetchInterval: 15000,
	});
}
