import { StatusBadge } from "@/components/StatusBadge";
import { Card, CardContent } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { useAgents } from "@/hooks/useAgents";
import { formatDistanceToNow } from "date-fns";

export function AgentsPage() {
	const { data: agents, isLoading } = useAgents();

	return (
		<div className="space-y-4">
			<h1 className="text-2xl font-semibold">Agents</h1>

			<Card>
				<CardContent className="p-0">
					{isLoading ? (
						<div className="space-y-2 p-6">
							{[...Array(3)].map((_, i) => (
								// biome-ignore lint/suspicious/noArrayIndexKey: skeleton
								<Skeleton key={i} className="h-10 w-full" />
							))}
						</div>
					) : !agents?.length ? (
						<p className="py-12 text-center text-sm text-muted-foreground">
							No agents registered.
						</p>
					) : (
						<table className="w-full text-sm">
							<thead>
								<tr className="border-b text-left text-muted-foreground">
									<th className="px-6 py-3 font-medium">Name</th>
									<th className="px-6 py-3 font-medium">Status</th>
									<th className="px-6 py-3 font-medium">Jobs</th>
									<th className="px-6 py-3 font-medium">Last Heartbeat</th>
									<th className="px-6 py-3 font-medium">Address</th>
									<th className="px-6 py-3 font-medium">Labels</th>
								</tr>
							</thead>
							<tbody>
								{agents.map((a) => (
									<tr key={a.id} className="border-b last:border-0 hover:bg-muted/50">
										<td className="px-6 py-3 font-medium font-mono text-xs">
											{a.name}
										</td>
										<td className="px-6 py-3">
											<StatusBadge status={a.status} />
										</td>
										<td className="px-6 py-3 text-xs text-muted-foreground">
											{a.running_jobs} / {a.capacity}
										</td>
										<td className="px-6 py-3 text-xs text-muted-foreground">
											{a.last_heartbeat_at
												? formatDistanceToNow(new Date(a.last_heartbeat_at), {
														addSuffix: true,
													})
												: "Never"}
										</td>
										<td className="px-6 py-3 font-mono text-xs text-muted-foreground">
											{a.address}
										</td>
										<td className="px-6 py-3">
											<div className="flex flex-wrap gap-1">
												{Object.entries(a.labels).map(([k, v]) => (
													<span
														key={k}
														className="font-mono text-xs bg-muted px-1.5 py-0.5 rounded"
													>
														{k}={v}
													</span>
												))}
											</div>
										</td>
									</tr>
								))}
							</tbody>
						</table>
					)}
				</CardContent>
			</Card>
		</div>
	);
}
