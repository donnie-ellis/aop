import { StatusBadge } from "@/components/StatusBadge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { useAgents } from "@/hooks/useAgents";
import { useJobs } from "@/hooks/useJobs";
import { formatDistanceToNow } from "date-fns";
import { Activity, CheckCircle2, Server, XCircle } from "lucide-react";
import { Link } from "react-router-dom";

function StatCard({
	title,
	value,
	icon: Icon,
	loading,
}: {
	title: string;
	value: number | string;
	icon: React.ElementType;
	loading?: boolean;
}) {
	return (
		<Card>
			<CardHeader className="flex flex-row items-center justify-between pb-2">
				<CardTitle className="text-sm font-medium text-muted-foreground">
					{title}
				</CardTitle>
				<Icon className="size-4 text-muted-foreground" />
			</CardHeader>
			<CardContent>
				{loading ? (
					<Skeleton className="h-8 w-16" />
				) : (
					<div className="text-2xl font-bold">{value}</div>
				)}
			</CardContent>
		</Card>
	);
}

export function DashboardPage() {
	const { data: jobs, isLoading: jobsLoading } = useJobs();
	const { data: agents, isLoading: agentsLoading } = useAgents();

	const running = jobs?.filter(
		(j) => j.status === "running" || j.status === "dispatched",
	) ?? [];
	const recent = [...(jobs ?? [])]
		.filter(
			(j) =>
				j.status === "success" ||
				j.status === "failed" ||
				j.status === "cancelled",
		)
		.sort(
			(a, b) =>
				new Date(b.updated_at).getTime() - new Date(a.updated_at).getTime(),
		)
		.slice(0, 10);

	const onlineAgents = agents?.filter((a) => a.status === "online").length ?? 0;
	const offlineAgents = agents?.filter((a) => a.status === "offline").length ?? 0;

	return (
		<div className="space-y-6">
			<h1 className="text-2xl font-semibold">Dashboard</h1>

			<div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
				<StatCard
					title="Running Jobs"
					value={running.length}
					icon={Activity}
					loading={jobsLoading}
				/>
				<StatCard
					title="Total Jobs"
					value={jobs?.length ?? 0}
					icon={CheckCircle2}
					loading={jobsLoading}
				/>
				<StatCard
					title="Agents Online"
					value={onlineAgents}
					icon={Server}
					loading={agentsLoading}
				/>
				<StatCard
					title="Agents Offline"
					value={offlineAgents}
					icon={XCircle}
					loading={agentsLoading}
				/>
			</div>

			{running.length > 0 && (
				<Card>
					<CardHeader>
						<CardTitle className="text-base">Active Jobs</CardTitle>
					</CardHeader>
					<CardContent className="p-0">
						<table className="w-full text-sm">
							<tbody>
								{running.map((job) => (
									<tr key={job.id} className="border-b last:border-0 hover:bg-muted/50">
										<td className="px-6 py-3">
											<Link
												to={`/jobs/${job.id}`}
												className="font-mono text-xs hover:underline"
											>
												{job.id.slice(0, 8)}
											</Link>
										</td>
										<td className="px-6 py-3">
											<StatusBadge status={job.status} />
										</td>
										<td className="px-6 py-3 text-muted-foreground text-xs">
											{formatDistanceToNow(new Date(job.created_at), {
												addSuffix: true,
											})}
										</td>
									</tr>
								))}
							</tbody>
						</table>
					</CardContent>
				</Card>
			)}

			<Card>
				<CardHeader>
					<CardTitle className="text-base">Recent Jobs</CardTitle>
				</CardHeader>
				<CardContent className="p-0">
					{jobsLoading ? (
						<div className="space-y-2 p-6">
							{[...Array(5)].map((_, i) => (
								// biome-ignore lint/suspicious/noArrayIndexKey: skeleton list
								<Skeleton key={i} className="h-8 w-full" />
							))}
						</div>
					) : recent.length === 0 ? (
						<p className="px-6 py-8 text-center text-sm text-muted-foreground">
							No completed jobs yet.
						</p>
					) : (
						<table className="w-full text-sm">
							<thead>
								<tr className="border-b text-left text-muted-foreground">
									<th className="px-6 py-3 font-medium">ID</th>
									<th className="px-6 py-3 font-medium">Status</th>
									<th className="px-6 py-3 font-medium">Finished</th>
								</tr>
							</thead>
							<tbody>
								{recent.map((job) => (
									<tr
										key={job.id}
										className="border-b last:border-0 hover:bg-muted/50"
									>
										<td className="px-6 py-3">
											<Link
												to={`/jobs/${job.id}`}
												className="font-mono text-xs hover:underline"
											>
												{job.id.slice(0, 8)}
											</Link>
										</td>
										<td className="px-6 py-3">
											<StatusBadge status={job.status} />
										</td>
										<td className="px-6 py-3 text-muted-foreground text-xs">
											{job.ended_at
												? formatDistanceToNow(new Date(job.ended_at), {
														addSuffix: true,
													})
												: "—"}
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
